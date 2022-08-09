#!/bin/python3
from collections import namedtuple
import base64, json, uuid, requests, hashlib, hmac
from pprint import pprint
from hashlib import scrypt, sha256 # TODO - audit! Should I use hazmat `Scrypt` instead for some reason?
import secrets

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

  @staticmethod
  def get_hash(wallet_id):
    response = requests.post('http://localhost:5279', json.dumps({
      "method": "sync_hash",
      "params": {
        "wallet_id": wallet_id,
      },
    }))
    return response.json()['result']

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
    self.API_VERSION = 3

    if local:
      BASE_URL = 'http://localhost:8090'
    else:
      BASE_URL = 'https://dev.lbry.id:8091'

    # Avoid confusion. I sometimes forget, at any rate.
    print ("Connecting to Wallet API at " + BASE_URL)

    API_URL = BASE_URL + '/api/%d' % self.API_VERSION

    self.AUTH_URL = API_URL + '/auth/full'
    self.REGISTER_URL = API_URL + '/signup'
    self.PASSWORD_URL = API_URL + '/password'
    self.WALLET_URL = API_URL + '/wallet'
    self.CLIENT_SALT_SEED_URL = API_URL + '/client-salt-seed'

  def register(self, email, password, salt_seed):
    body = json.dumps({
      'email': email,
      'password': password,
      'clientSaltSeed': salt_seed,
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

  def get_salt_seed(self, email):
    params = {
      'email': base64.encodestring(bytes(email.encode('utf-8'))),
    }
    response = requests.get(self.CLIENT_SALT_SEED_URL, params=params)

    if response.status_code == 404:
      print ('Account not found')
      raise Exception("Account not found")

    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      raise Exception("Unexpected status code")

    return response.json()['clientSaltSeed']

  def get_wallet(self, token):
    params = {
      'token': token,
    }
    response = requests.get(self.WALLET_URL, params=params)

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
      "token": token,
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

  def change_password_with_wallet(self, wallet_state, hmac, email, old_password, new_password, salt_seed):
    body = json.dumps({
      "encryptedWallet": wallet_state.encrypted_wallet,
      "sequence": wallet_state.sequence,
      "hmac": hmac,
      "email": email,
      "oldPassword": old_password,
      "newPassword": new_password,
      'clientSaltSeed': salt_seed,
    })

    response = requests.post(self.PASSWORD_URL, body)

    if response.status_code == 200:
      print ('Successfully updated password and wallet state on server')
      return True
    elif response.status_code == 409:
      print ('Either submitted wallet is out of date, or there was no wallet on the server to update in the first place.')
      return False
    else:
      print ('Error', response.status_code)
      print (response.content)
      raise Exception("Unexpected status code")

  def change_password_no_wallet(self, email, old_password, new_password, salt_seed):
    body = json.dumps({
      "email": email,
      "oldPassword": old_password,
      "newPassword": new_password,
      'clientSaltSeed': salt_seed,
    })

    response = requests.post(self.PASSWORD_URL, body)

    if response.status_code == 200:
      print ('Successfully updated password on server')
      return True
    elif response.status_code == 409:
      print ('There is a wallet on the server that needs to be updated with password change.')
      return False
    else:
      print ('Error', response.status_code)
      print (response.content)
      raise Exception("Unexpected status code")

# Thanks to Standard Notes. See:
# https://docs.standardnotes.com/specification/encryption/

# Sized in bytes
SALT_SEED_LENGTH = 32
SALT_LENGTH = 16

def generate_salt_seed():
  return secrets.token_hex(SALT_SEED_LENGTH)

def generate_salt(email, seed):
  hash_input = (email + ":" + seed).encode('utf-8')
  hash_output = sha256(hash_input).hexdigest().encode('utf-8')
  return bytes(hash_output[:(SALT_LENGTH * 2)])

def derive_secrets(root_password, email, salt_seed):
    # 2017 Scrypt parameters: https://words.filippo.io/the-scrypt-parameters/
    #
    # There's recommendations for interactive use, and stronger recommendations
    # for sensitive storage. Going with the latter since we're storing
    # encrypted stuff on a server.
    #
    # Auditors double check.
    scrypt_n = 1<<20
    scrypt_r = 8
    scrypt_p = 1

    key_length = 32
    num_keys = 2

    salt = generate_salt(email, salt_seed)

    print ("Generating keys...")
    kdf_output = scrypt(
      bytes(root_password, 'utf-8'),
      salt=salt,
      dklen=key_length * num_keys,
      n=scrypt_n,
      r=scrypt_r,
      p=scrypt_p,
      maxmem=1100000000, # TODO - is this a lot?
    )
    print ("Done generating keys")

    # Split the output in three
    parts = (
      kdf_output[:key_length],
      kdf_output[key_length:key_length * 2],
    )

    lbry_id_password = base64.b64encode(parts[0]).decode('utf-8')
    hmac_key = parts[1]

    return lbry_id_password, hmac_key

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
    self.wallet_sync_api = WalletSync(local=local)

    # Represents normal client behavior (though a real client will of course save device id)
    self.device_id = str(uuid.uuid4())
    self.auth_token = 'bad token'
    self.synced_wallet_state = None

    self.email = email
    self.root_password = root_password

    self.wallet_id = wallet_id

  def register(self):
    # Note that for each registration, i.e. for each domain, we generate a
    # different salt seed.
    #
    # Auditor - Does changing salt seed here cover the threat of sync servers
    # guessing the password of the same user on another sync server? It should
    # be a new seed if it's a new server.

    self.salt_seed = generate_salt_seed()
    self.lbry_id_password, self.hmac_key = derive_secrets(
      self.root_password, self.email, self.salt_seed)

    success = self.wallet_sync_api.register(
      self.email,
      self.lbry_id_password,
      self.salt_seed
    )
    if success:
      print ("Registered")

  def set_local_password(self, root_password):
    """
    For clients to catch up to another client that just changed the password.
    """
    # TODO - is UTF-8 appropriate for root_password? based on characters used etc.
    self.root_password = root_password
    self.update_derived_secrets()

  def update_derived_secrets(self):
    """
    For clients other than the one that most recently registered or changed the
    password, use this to get the salt seed from the server and generate keys
    locally.
    """
    self.salt_seed = self.wallet_sync_api.get_salt_seed(self.email)
    self.lbry_id_password, self.hmac_key = derive_secrets(
      self.root_password, self.email, self.salt_seed)

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

      # TODO - This should be the encrypted form of the empty wallet. The very
      # first baseline, which could be used for merges in weird cases where
      # users make conflicting changes on two different clients before ever
      # pushing to the sync server.
      encrypted_wallet="",
    )
    # Initialize to the hash of the empty wallet. This way we will know if any
    # changes to the wallet exist that haven't been pushed yet, even if the
    # changes were made before the wallet state was initialized.
    # TODO - actually set the right hash
    self.mark_local_changes_synced_to_empty()

  def get_auth_token(self):
    token = self.wallet_sync_api.get_auth_token(
      self.email,
      self.lbry_id_password,
      self.device_id,
    )
    if not token:
      # In a real client, this is where you may consider
      # a) Offering to have the user change their password
      # b) Try update_derived_secrets() and get_auth_token() silently, for the unlikely case that the user changed their password back and forth
      print ("Failed to get the auth token. Do you need to update this client's password (set_local_password())?")
      print ("Or, in the off-chance the user changed their password back and forth, try updating secrets (update_derived_secrets()) to get the latest salt seed.")
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
    # the above eventually) so we will just return `new_wallet_state`. However,
    # since we can at least compare hashes, we'll leave a little note for the
    # user indicating that we're doing a merge. Caveat: We can't do it on
    # sequence=0 because we can't get a sense of whether changes were made on a
    # client before the first sync.
    if self.synced_wallet_state.sequence > 0:
      if self.has_unsynced_local_changes():
        print ("Merging local changes with remote changes to create latest walletState.")
      else:
        print ("Nothing to merge. Taking remote walletState as latest walletState.")
    return new_wallet_state

  # Returns: status
  def get_remote_wallet(self):
    # TODO - Do try/catch for other calls I guess. I needed it here in
    # particular for the README
    try:
      new_wallet_state, hmac = self.wallet_sync_api.get_wallet(self.auth_token)
    except Exception:
      return "Failed to get remote wallet"

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

    # We just took the value from the sync server, so local changes are synced
    self.mark_local_changes_synced()

    print ("Got latest walletState:")
    pprint(self.synced_wallet_state)
    return "Success"

  # Returns: status
  def update_remote_wallet(self):
    # Create a *new* wallet state, with the updated sequence, and include our
    # local encrypted wallet changes. Don't set self.synced_wallet_state to
    # this until we know that it's accepted by the server.
    if not self.synced_wallet_state:
      print ("No wallet state to post.")
      return "Error"

    submitted_wallet_state = WalletState(
      encrypted_wallet=self.get_local_encrypted_wallet(self.root_password),
      sequence=self.synced_wallet_state.sequence + 1
    )
    hmac = create_hmac(submitted_wallet_state, self.hmac_key)

    # Submit our wallet.
    updated = self.wallet_sync_api.update_wallet(submitted_wallet_state, hmac, self.auth_token)

    if updated:
      # We updated it. Now it's synced and we mark it as such.
      self.synced_wallet_state = submitted_wallet_state

      # We just pushed our local changes to the server, so local changes are synced
      self.mark_local_changes_synced()

      print ("Synced walletState:")
      pprint(self.synced_wallet_state)
      return "Success"

    print ("Could not update. Need to get new wallet and merge")
    return "Failure"

  # Returns: status
  def change_password(self, new_root_password):
    # Change the password on the server. If a wallet exists on the server,
    # update that as well so that the sync password and hmac key are derived
    # from the same root password as the lbry id password.

    # Auditor - Should we be generating a *new* seed for every password change?
    self.salt_seed = generate_salt_seed()
    new_lbry_id_password, new_hmac_key = derive_secrets(
      new_root_password, self.email, self.salt_seed)
    def set_secrets():
      # Only do this once we got a good response from the server.
      # In a function because it can happen in two different places.
      self.root_password, self.lbry_id_password, self.hmac_key = (
        new_root_password, new_lbry_id_password, new_hmac_key)

    # TODO - Think of failure sequence in case of who knows what. We
    # could just get the old salt seed back from the server?
    # We can't lose it though. Keep the old one around? Kinda sucks.

    if self.synced_wallet_state and self.synced_wallet_state.sequence > 0:
      # Don't allow it to change if we have local changes to push. This
      # precludes the possibility of having a conflict with remote changes,
      # followed by a merge with user interaction, when the user is already in
      # the middle of a password change. This way, if there is a conflict, we
      # can simply get the latest wallet and try again with the same password
      # that the user just entered, guaranteeing that they won't need to do any
      # more interactions.
      #
      # NOTE: If for whatever reason this is removed, make sure to add a call
      # to mark_local_changes_synced as appropriate below, since we may be
      # going from unsynced to synced.
      if self.has_unsynced_local_changes():
        print("Local changes found. Update remote wallet before changing password.")
        return "Failure"

      # Create a *new* wallet state (with our new sync password), with the
      # updated sequence, and include our local encrypted wallet changes.
      # Don't set self.synced_wallet_state to this until we know that it's
      # accepted by the server.

      submitted_wallet_state = WalletState(
        encrypted_wallet=self.get_local_encrypted_wallet(new_root_password),
        sequence=self.synced_wallet_state.sequence + 1
      )
      hmac = create_hmac(submitted_wallet_state, new_hmac_key)

      # Update our password and submit our wallet.
      updated = self.wallet_sync_api.change_password_with_wallet(submitted_wallet_state, hmac, self.email, self.lbry_id_password, new_lbry_id_password, self.salt_seed)

      if updated:
        # We updated it. Now it's synced and we mark it as such. Update everything at once to keep local changes in sync!
        self.synced_wallet_state = submitted_wallet_state
        set_secrets()

        print ("Synced walletState:")
        pprint(self.synced_wallet_state)
        return "Success"
    else:
      # Update our password.
      updated = self.wallet_sync_api.change_password_no_wallet(self.email, self.lbry_id_password, new_lbry_id_password, self.salt_seed)

      if updated:
        # We updated it. Now we mark it as such. Update everything at once to keep local changes in sync!
        set_secrets()
        return "Success"

    print ("Could not update wallet and password. Perhaps need to get new wallet and merge, perhaps something else.")
    return "Failure"

  def set_preference(self, key, value):
    # TODO - error checking
    return LBRYSDK.set_preference(self.wallet_id, key, value)

  def get_preferences(self):
    # TODO - error checking
    return LBRYSDK.get_preferences(self.wallet_id)

  def has_unsynced_local_changes(self):
    return self.lbry_sdk_last_synced_hash != LBRYSDK.get_hash(self.wallet_id)

  def mark_local_changes_synced(self):
    self.lbry_sdk_last_synced_hash = LBRYSDK.get_hash(self.wallet_id)

  def mark_local_changes_synced_to_empty(self):
    # TODO - this should be the hash of the empty wallet. See
    # comment in init_wallet_state().
    self.lbry_sdk_last_synced_hash = ""

  def update_local_encrypted_wallet(self, encrypted_wallet):
    # TODO - error checking
    return LBRYSDK.update_wallet(self.wallet_id, self.root_password, encrypted_wallet)

  def get_local_encrypted_wallet(self, sync_password):
    # Note for auditor: sync_password here is now the root_password. The SDK
    # has its own KDF (though with different Scrypt parameters as of this
    # writing). So in all:
    # root password -> APP KDF -> (HMAC, wallet sync server password)
    # root password -> SDK KDF -> (wallet encryption for remote storage, wallet "locking" (encryption) for local storage)
    # The App uses the Salt Seed system from Standard Notes, the SDK creates a
    # random salt every encryption. So (for now) we're not sharing salts
    # between the KDFs. The question is, is it safe to use the same root
    # password on two two different KDFs like this?

    # TODO - error checking
    return LBRYSDK.get_wallet(self.wallet_id, sync_password)
