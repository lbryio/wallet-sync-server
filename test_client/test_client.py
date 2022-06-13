#!/bin/python3
from collections import namedtuple
import random, string, json, uuid, requests, hashlib
from pprint import pprint

CURRENT_VERSION = 1

WalletState = namedtuple('WalletState', ['sequence', 'encrypted_wallet'])

class WalletSync():
  BASE_URL = 'http://localhost:8090'
  AUTH_URL = BASE_URL + '/auth/full'
  REGISTER_URL = BASE_URL + '/signup'
  WALLET_URL = BASE_URL + '/wallet'

  @classmethod
  def register(cls, email, password):
    body = json.dumps({
      'email': email,
      'password': password,
    })
    response = requests.post(cls.REGISTER_URL, body)
    if response.status_code != 201:
      print ('Error', response.status_code)
      print (response.content)
      return False
    return True

  @classmethod
  def get_auth_token(cls, email, password, device_id):
    body = json.dumps({
      'email': email,
      'password': password,
      'deviceId': device_id,
    })
    response = requests.post(cls.AUTH_URL, body)
    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      return None

    return response.json()['token']

  @classmethod
  def get_wallet(cls, token):
    params = {
      'token': token,
    }
    response = requests.get(cls.WALLET_URL, params=params)

    # TODO check response version on client side now

    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      return None, None

    wallet_state = WalletState(
      encrypted_wallet=response.json()['encryptedWallet'],
      sequence=response.json()['sequence'],
    )
    hmac = response.json()['hmac']
    return wallet_state, hmac

  @classmethod
  def update_wallet(cls, wallet_state, hmac, token):
    body = json.dumps({
      'version': CURRENT_VERSION,
      'token': token,
      "encryptedWallet": wallet_state.encrypted_wallet,
      "sequence": wallet_state.sequence,
      "hmac": hmac,
    })

    response = requests.post(cls.WALLET_URL, body)

    # TODO check that response.json().version == CURRENT_VERSION

    if response.status_code == 200:
      conflict = False
      print ('Successfully updated wallet state on server')
    elif response.status_code == 409:
      conflict = True
      print ('Wallet state out of date. Getting updated wallet state. Try posting again after this.')
      # Not an error! We still want to merge in the returned wallet.
    else:
      print ('Error', response.status_code)
      print (response.content)
      return None, None, None

    wallet_state = WalletState(
      encrypted_wallet=response.json()['encryptedWallet'],
      sequence=response.json()['sequence'],
    )
    hmac = response.json()['hmac']
    return wallet_state, hmac, conflict

# TODO - do this correctly. This is a hack example.
def derive_login_password(root_password):
    return hashlib.sha256('login:' + root_password.encode('utf-8')).hexdigest()

# TODO - do this correctly. This is a hack example.
def derive_sdk_password(root_password):
    return hashlib.sha256('sdk:' + root_password.encode('utf-8')).hexdigest()

# TODO - do this correctly. This is a hack example.
def derive_hmac_key(root_password):
    return hashlib.sha256('hmac:' + root_password.encode('utf-8')).hexdigest()

# TODO - do this correctly. This is a hack example.
def create_hmac(wallet_state, hmac_key):
    input_str = hmac_key + ':' + str(wallet_state.sequence) + ':' + wallet_state.encrypted_wallet
    return hashlib.sha256(input_str.encode('utf-8')).hexdigest()

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

  def __init__(self):
    # Represents normal client behavior (though a real client will of course save device id)
    self.device_id = str(uuid.uuid4())

    self.auth_token = 'bad token'

    self.synced_wallet_state = None

    # TODO - save change to disk in between, associated with account and/or
    # wallet
    self._encrypted_wallet_local_changes = ''

  # TODO - make this act more sdk-like. in fact maybe even install the sdk?

  # TODO - This does not deal with the question of tying accounts to wallets.
  # Does a new wallet state mean a we're creating a new account? What happens
  # if we create a new wallet state tied to an existing account? Do we merge it
  # with what's on the server anyway? Do we refuse to merge, or warn the user?
  # Etc. This sort of depends on how the LBRY Desktop/SDK usually behave. For
  # now, it'll end up just merging any un-saved local changes with whatever is
  # on the server.
  def new_wallet_state(self):
    # Represents what's been synced to the wallet sync server. It starts with
    # sequence=0 which means nothing has been synced yet.
    self.synced_wallet_state = WalletState(sequence=0, encrypted_wallet='-')

    # TODO - actual encryption with encryption_key - or maybe not.
    self._encrypted_wallet_local_changes = ''

  def set_account(self, email, root_password):
    self.email = email
    self.root_password = root_password

  def register(self):
    success = WalletSync.register(
      self.email,
      derive_login_password(self.root_password),
    )
    if success:
      print ("Registered")

  def get_auth_token(self):
    token = WalletSync.get_auth_token(
      self.email,
      derive_login_password(self.root_password),
      self.device_id,
    )
    if not token:
      return
    self.auth_token = token
    print ("Got auth token: ", self.auth_token)

  # TODO - What about cases where we are managing multiple different wallets?
  # Some will have lower sequences. If you accidentally mix it up client-side,
  # you might end up overwriting one with a lower sequence entirely. Maybe we
  # want to annotate them with which account we're talking about. Again, we
  # should see how LBRY Desktop/SDK deal with it.
  def get_wallet(self):
    new_wallet_state, hmac = WalletSync.get_wallet(self.auth_token)

    # If there was a failure
    if not new_wallet_state:
      return

    hmac_key = derive_hmac_key(self.root_password)
    if not check_hmac(new_wallet_state, hmac_key, hmac):
      print ('Error - bad hmac on new wallet')
      print (new_wallet_state, hmac)
      return

    if self.synced_wallet_state != new_wallet_state and not self._validate_new_wallet_state(new_wallet_state):
      print ('Error - new wallet does not validate')
      print ('current:', self.synced_wallet_state)
      print ('got:', new_wallet_state)
      return

    if self.synced_wallet_state is None:
      # This is if we're getting a wallet_state for the first time. Initialize
      # the local changes.
      self._encrypted_wallet_local_changes = ''

    self.synced_wallet_state = new_wallet_state

    print ("Got latest walletState:")
    pprint(self.synced_wallet_state)

  def update_wallet(self):
    # Create a *new* wallet state, indicating that it was last updated by this
    # device, with the updated sequence, and include our local encrypted wallet changes.
    # Don't set self.synced_wallet_state to this until we know that it's accepted by
    # the server.
    if not self.synced_wallet_state:
      print ("No wallet state to post.")
      return

    hmac_key = derive_hmac_key(self.root_password)

    submitted_wallet_state = WalletState(
      encrypted_wallet=self.cur_encrypted_wallet(),
      sequence=self.synced_wallet_state.sequence + 1
    )
    hmac = create_hmac(submitted_wallet_state, hmac_key)

    # Submit our wallet, get the latest wallet back as a response
    new_wallet_state, new_hmac, conflict = WalletSync.update_wallet(submitted_wallet_state, hmac, self.auth_token)

    # If there was a failure (not just a conflict)
    if not new_wallet_state:
      return

    # TODO - there's some code in common here with the get_wallet function. factor it out.

    if not check_hmac(new_wallet_state, hmac_key, new_hmac):
      print ('Error - bad hmac on new wallet')
      print (new_wallet_state, hmac)
      return

    if submitted_wallet_state != new_wallet_state and not self._validate_new_wallet_state(new_wallet_state):
      print ('Error - new wallet does not validate')
      print ('current:', self.synced_wallet_state)
      print ('got:', new_wallet_state)
      return

    # If there's not a conflict, we submitted successfully and should reset our previously local changes
    if not conflict:
      self._encrypted_wallet_local_changes = ''

    self.synced_wallet_state = new_wallet_state

    print ("Got new walletState:")
    pprint(self.synced_wallet_state)

  def change_encrypted_wallet(self):
    if not self.synced_wallet_state:
      print ("No wallet state, so we can't add to it yet.")
      return

    self._encrypted_wallet_local_changes += ':' + ''.join(random.choice(string.hexdigits) for x in range(4))

  def cur_encrypted_wallet(self):
    if not self.synced_wallet_state:
      print ("No wallet state, so no encrypted wallet.")
      return

    # The local changes on top of whatever came from the server
    # If we pull new changes from server, we "rebase" these on top of it
    # If we push changes, the full "rebased" version gets committed to the server
    return self.synced_wallet_state.encrypted_wallet + self._encrypted_wallet_local_changes
