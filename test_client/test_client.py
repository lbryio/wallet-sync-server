#!/bin/python3
from collections import namedtuple
import base64, json, uuid, requests, hashlib, hmac
from pprint import pprint
from hashlib import scrypt # TODO - audit! Should I use hazmat `Scrypt` instead for some reason?

WalletState = namedtuple('WalletState', ['sequence', 'encrypted_wallet'])

class LBRYSDK():
  # TODO - error checking
  @staticmethod
  def get_wallet(wallet_id, password):
    response = requests.post('http://localhost:5279', json.dumps({
      "method": "sync_apply",
      "params": {
        "password": password,
        "wallet_id": wallet_id,
      },
    }))
    return response.json()['result']['data']

  # TODO - error checking
  @staticmethod
  def update_wallet(wallet_id, password, data):
    response = requests.post('http://localhost:5279', json.dumps({
      "method": "sync_apply",
      "params": {
        "data": data,
        "password": password,
        "wallet_id": wallet_id,
      },
    }))
    return response.json()['result']['data']

  # TODO - error checking
  @staticmethod
  def set_preference(wallet_id, key, value):
    response = requests.post('http://localhost:5279', json.dumps({
      "method": "preference_set",
      "params": {
        "key": key,
        "value": value,
        "wallet_id": wallet_id,
      },
    }))
    return response.json()['result']

  # TODO - error checking
  @staticmethod
  def get_preferences(wallet_id):
    response = requests.post('http://localhost:5279', json.dumps({
      "method": "preference_get",
      "params": {
        "wallet_id": wallet_id,
      },
    }))
    return response.json()['result']

class WalletSync():
  def __init__(self, local):
    self.API_VERSION = 2

    if local:
      BASE_URL = 'http://localhost:8090'
    else:
      BASE_URL = 'https://dev.lbry.id:8091'
    API_URL = BASE_URL + '/api/%d' % self.API_VERSION

    self.AUTH_URL = API_URL + '/auth/full'
    self.REGISTER_URL = API_URL + '/signup'
    self.WALLET_URL = API_URL + '/wallet'

  def register(self, email, password):
    body = json.dumps({
      'email': email,
      'password': password,
    })
    response = requests.post(self.REGISTER_URL, body)
    if response.status_code != 201:
      print ('Error', response.status_code)
      print (response.content)
      return False
    return True

  def get_auth_token(self, email, password, device_id):
    body = json.dumps({
      'email': email,
      'password': password,
      'deviceId': device_id,
    })
    response = requests.post(self.AUTH_URL, body)
    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      return None

    return response.json()['token']

  def get_wallet(self, token):
    params = {
      'token': token,
    }
    response = requests.get(self.WALLET_URL, params=params)

    # TODO check response version on client side now
    if response.status_code == 404:
      print ('Wallet not found')
      # "No wallet" is not an error, so no exception raised
      return None, None

    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      raise Exception("Unexpected status code")

    wallet_state = WalletState(
      encrypted_wallet=response.json()['encryptedWallet'],
      sequence=response.json()['sequence'],
    )
    hmac = response.json()['hmac']
    return wallet_state, hmac

  def update_wallet(self, wallet_state, hmac, token):
    body = json.dumps({
      'token': token,
      "encryptedWallet": wallet_state.encrypted_wallet,
      "sequence": wallet_state.sequence,
      "hmac": hmac,
    })

    response = requests.post(self.WALLET_URL, body)

    if response.status_code == 200:
      print ('Successfully updated wallet state on server')
      return True
    elif response.status_code == 409:
      print ('Submitted wallet is out of date.')
      return False
    else:
      print ('Error', response.status_code)
      print (response.content)
      raise Exception("Unexpected status code")

def derive_secrets(root_password, salt):
    # TODO - Audit me audit me audit me! I don't know if these values are
    # optimal.
    #
    # TODO - wallet_id in the salt? (with domain etc if we go that way)
    # But, we probably want random salt anyway for each domain, who cares
    #
    # TODO - save scrypt parameters with the keys so we can change parameters
    # and still read old keys?
    #
    # https://stackoverflow.com/a/12581268
    # Per this, there's an optimal for interactive use, and there's a stronger
    # optimal for sensitive storage. Going with the latter since we're storing
    # encrypted stuff on a server. That said, that's based on presentation
    # slides from 2009. Maybe I should go even more secure?
    scrypt_n = 1<<20
    scrypt_r = 8
    scrypt_p = 1

    key_length = 32
    num_keys = 3

    kdf_output = scrypt(
      bytes(root_password, 'utf-8'),
      salt=salt,
      dklen=key_length * num_keys,
      n=scrypt_n,
      r=scrypt_r,
      p=scrypt_p,
      maxmem=1100000000, # TODO - is this a lot?
    )

    # Split the output in three
    parts = (
      kdf_output[:key_length],
      kdf_output[key_length:key_length * 2],
      kdf_output[key_length * 2:],
    )

    lbry_id_password = base64.b64encode(parts[0]).decode('utf-8')
    sync_password = base64.b64encode(parts[1]).decode('utf-8')
    hmac_key = parts[2]

    return lbry_id_password, sync_password, hmac_key

def create_hmac(wallet_state, hmac_key):
    input_str = str(wallet_state.sequence) + ':' + wallet_state.encrypted_wallet
    return hmac.new(hmac_key, input_str.encode('utf-8'), hashlib.sha256 ).hexdigest()

def check_hmac(wallet_state, hmac_key, hmac):
    return hmac == create_hmac(wallet_state, hmac_key)

class Client():
  # If you want to get the lastSynced stuff back, see:
  # 512ebe3e95bf4e533562710a7f91c59616a9a197
  # It's mostly simple, but the _validate_new_wallet_state changes may be worth
  # looking at.
  def _validate_new_wallet_state(self, new_wallet_state):
    if self.synced_wallet_state is None:
      # All of the validations here are in reference to what the device already
      # has. If this device is getting a wallet state for the first time, there
      # is no basis for comparison.
      return True

    # Make sure that the new sequence is overall later.
    if new_wallet_state.sequence <= self.synced_wallet_state.sequence:
      return False

    return True

  def __init__(self, email, root_password, wallet_id='default_wallet', local=False):
    # Represents normal client behavior (though a real client will of course save device id)
    self.device_id = str(uuid.uuid4())
    self.auth_token = 'bad token'
    self.synced_wallet_state = None

    self.email = email

    # TODO - generate randomly CLIENT SIDE and post to server with
    # registration. And maybe get it to new clients along with the auth token.
    # But is there an attack vector if we don't trust the salt? See how others
    # do it. Since the same server sees one of the outputs of the KDF. Huh.
    self.salt = b'I AM A SALT'

    # TODO - is UTF-8 appropriate for root_password? based on characters used etc.
    self.lbry_id_password, self.sync_password, self.hmac_key = derive_secrets(root_password, self.salt)

    self.wallet_id = wallet_id

    self.wallet_sync_api = WalletSync(local=local)

  # TODO - This does not deal with the question of tying accounts to wallets.
  # Does a new wallet state mean a we're creating a new account? What happens
  # if we create a new wallet state tied to an existing account? Do we merge it
  # with what's on the server anyway? Do we refuse to merge, or warn the user?
  # Etc. This sort of depends on how the LBRY Desktop/SDK usually behave. For
  # now, it'll end up just merging any un-saved local changes with whatever is
  # on the server.

  # TODO - Later, we should be saving the synced_wallet_state to disk, or
  # something like that, so we know whether there are unpushed changes on
  # startup (which should be uncommon but possible if crash or network problem
  # in previous run). This will be important when the client is responsible for
  # merging what comes from the server with those local unpushed changes. For
  # now, the SDK handles merges with timestamps and such so it's as safe as
  # always to just merge in.

  # TODO - Save wallet state to disk, and init by pulling from disk. That way,
  # we'll know what the merge base is, and we won't have to merge from 0 each
  # time the app restarts.

  # TODO - Wrap this back into __init__, now that I got the empty encrypted
  # wallet right.
  def init_wallet_state(self):
    # Represents what's been synced to the wallet sync server. It starts with
    # sequence=0 which means nothing has been synced yet. As such, we start
    # with an empty encrypted_wallet here. Anything currently in the SDK is a
    # local-only change until it's pushed. If there's a merge conflict,
    # sequence=0, empty encrypted_wallet will be the merge base. That way we
    # won't lose any changes.
    self.synced_wallet_state = WalletState(
      sequence=0,
      encrypted_wallet="",
    )

  def register(self):
    success = self.wallet_sync_api.register(
      self.email,
      self.lbry_id_password,
    )
    if success:
      print ("Registered")

  def get_auth_token(self):
    token = self.wallet_sync_api.get_auth_token(
      self.email,
      self.lbry_id_password,
      self.device_id,
    )
    if not token:
      return
    self.auth_token = token
    print ("Got auth token: ", self.auth_token)

  # TODO - What about cases where we are managing multiple different wallets?
  # Some will have lower sequences. If you accidentally mix it up client-side,
  # you might end up overwriting one wallet with another if the former has a
  # higher sequence number. Maybe we want to annotate them with which account
  # we're talking about. Again, we should see how LBRY Desktop/SDK deal with
  # it.

  def get_merged_wallet_state(self, new_wallet_state):
    # Eventually, we will look for local changes in
    # `get_local_encrypted_wallet()` by comparing it to
    # `self.synced_wallet_state.encrypted_wallet`.
    #
    # If there are no local changes, we can just return `new_wallet_state`.
    #
    # If there are local changes, we will merge between `new_wallet_state` and
    # `get_local_encrypted_wallet()`, using
    # `self.synced_wallet_state.encrypted_wallet` as our merge base.
    #
    # For really hairy cases, this could even be a whole interactive process,
    # not just a function.

    # For now, the SDK handles merging (in a way that we hope to improve with
    # the above eventually) so we will just return `new_wallet_state`.
    #
    # It would be nice to have a little "we just merged in changes" log output
    # if there are local changes, just for demo purpoes. Unfortunately, the SDK
    # outputs a different encrypted blob each time we ask it for the encrypted
    # wallet, so there's no easy way to check if it actually changed.
    return new_wallet_state

  # Returns: status
  def get_remote_wallet(self):
    new_wallet_state, hmac = self.wallet_sync_api.get_wallet(self.auth_token)

    if not new_wallet_state:
      # Wallet not found, but this is not an error
      return "Not Found"

    if not check_hmac(new_wallet_state, self.hmac_key, hmac):
      print ('Error - bad hmac on new wallet')
      print (new_wallet_state, hmac)
      return "Error"

    if self.synced_wallet_state != new_wallet_state and not self._validate_new_wallet_state(new_wallet_state):
      print ('Error - new wallet does not validate')
      print ('current:', self.synced_wallet_state)
      print ('got:', new_wallet_state)
      return "Error"

    merged_wallet_state = self.get_merged_wallet_state(new_wallet_state)

    # TODO error recovery between these two steps? sequence of events?
    # This isn't gonna be quite right. Look at state diagrams.
    self.synced_wallet_state = merged_wallet_state
    self.update_local_encrypted_wallet(merged_wallet_state.encrypted_wallet)

    print ("Got (and maybe merged in) latest walletState:")
    pprint(self.synced_wallet_state)
    return "Success"

  # Returns: status
  def update_remote_wallet(self):
    # Create a *new* wallet state, indicating that it was last updated by this
    # device, with the updated sequence, and include our local encrypted wallet changes.
    # Don't set self.synced_wallet_state to this until we know that it's accepted by
    # the server.
    if not self.synced_wallet_state:
      print ("No wallet state to post.")
      return "Error"

    submitted_wallet_state = WalletState(
      encrypted_wallet=self.get_local_encrypted_wallet(),
      sequence=self.synced_wallet_state.sequence + 1
    )
    hmac = create_hmac(submitted_wallet_state, self.hmac_key)

    # Submit our wallet.
    updated = self.wallet_sync_api.update_wallet(submitted_wallet_state, hmac, self.auth_token)

    if updated:
      # We updated it. Now it's synced and we mark it as such.
      self.synced_wallet_state = submitted_wallet_state

      print ("Synced walletState:")
      pprint(self.synced_wallet_state)
      return "Success"

    print ("Could not update. Need to get new wallet and merge")
    return "Failure"

  def set_preference(self, key, value):
    # TODO - error checking
    return LBRYSDK.set_preference(self.wallet_id, key, value)

  def get_preferences(self):
    # TODO - error checking
    return LBRYSDK.get_preferences(self.wallet_id)

  def update_local_encrypted_wallet(self, encrypted_wallet):
    # TODO - error checking
    return LBRYSDK.update_wallet(self.wallet_id, self.sync_password, encrypted_wallet)

  def get_local_encrypted_wallet(self):
    # TODO - error checking
    return LBRYSDK.get_wallet(self.wallet_id, self.sync_password)
