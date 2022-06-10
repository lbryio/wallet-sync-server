#!/bin/python3
from collections import namedtuple
import random, string, json, uuid, requests, hashlib
from pprint import pprint

CURRENT_VERSION = 1

BASE_URL = 'http://localhost:8090'
AUTH_URL = BASE_URL + '/auth/full'
REGISTER_URL = BASE_URL + '/signup'
WALLET_URL = BASE_URL + '/wallet'

# TODO - We should have:
# * self.last_synced_wallet_state - as described
# * self.current_wallet_state - WalletState(cur_encrypted_wallet(), sequence + 1) - and current_wallet_state
# We don't need it yet but we'd be avoiding the entire point of the syncing system. At least keep it around in this demo.

WalletState = namedtuple('WalletState', ['sequence', 'encrypted_wallet'])

# TODO - do this correctly. This is a hack example.
def derive_login_password(root_password):
    return hashlib.sha256(root_password.encode('utf-8')).hexdigest()[:10]

# TODO - do this correctly. This is a hack example.
def derive_sdk_password(root_password):
    return hashlib.sha256(root_password.encode('utf-8')).hexdigest()[10:20]

# TODO - do this correctly. This is a hack example.
def derive_hmac_key(root_password):
    return hashlib.sha256(root_password.encode('utf-8')).hexdigest()[20:]

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
    if self.wallet_state is None:
      # All of the validations here are in reference to what the device already
      # has. If this device is getting a wallet state for the first time, there
      # is no basis for comparison.
      return True

    # Make sure that the new sequence is overall later.
    if new_wallet_state.sequence <= self.wallet_state.sequence:
      return False

    return True

  def __init__(self):
    # Represents normal client behavior (though a real client will of course save device id)
    self.device_id = str(uuid.uuid4())

    self.auth_token = 'bad token'

    self.wallet_state = None

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
    # camel-cased to ease json interop
    self.wallet_state = WalletState(sequence=0, encrypted_wallet='-')

    # TODO - actual encryption with encryption_key - or maybe not.
    self._encrypted_wallet_local_changes = ''

  def set_account(self, email, root_password):
    self.email = email
    self.root_password = root_password

  def register(self):
    body = json.dumps({
      'email': self.email,
      'password': derive_login_password(self.root_password),
    })
    response = requests.post(REGISTER_URL, body)
    if response.status_code != 201:
      print ('Error', response.status_code)
      print (response.content)
      return
    print ("Registered")

  def get_auth_token(self):
    body = json.dumps({
      'email': self.email,
      'password': derive_login_password(self.root_password),
      'deviceId': self.device_id,
    })
    response = requests.post(AUTH_URL, body)
    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      return
    self.auth_token = json.loads(response.content)['token']
    print ("Got auth token: ", self.auth_token)

  # TODO - What about cases where we are managing multiple different wallets?
  # Some will have lower sequences. If you accidentally mix it up client-side,
  # you might end up overwriting one with a lower sequence entirely. Maybe we
  # want to annotate them with which account we're talking about. Again, we
  # should see how LBRY Desktop/SDK deal with it.
  def get_wallet(self):
    params = {
      'token': self.auth_token,
    }
    response = requests.get(WALLET_URL, params=params)
    if response.status_code != 200:
      # TODO check response version on client side now
      print ('Error', response.status_code)
      print (response.content)
      return

    hmac_key = derive_hmac_key(self.root_password)

    new_wallet_state = WalletState(
      encrypted_wallet=response.json()['encryptedWallet'],
      sequence=response.json()['sequence'],
    )
    hmac = response.json()['hmac']
    if not check_hmac(new_wallet_state, hmac_key, hmac):
      print ('Error - bad hmac on new wallet')
      print (response.content)
      return

    if self.wallet_state != new_wallet_state and not self._validate_new_wallet_state(new_wallet_state):
      print ('Error - new wallet does not validate')
      print (response.content)
      return

    if self.wallet_state is None:
      # This is if we're getting a wallet_state for the first time. Initialize
      # the local changes.
      self._encrypted_wallet_local_changes = ''

    self.wallet_state = new_wallet_state

    print ("Got latest walletState:")
    pprint(self.wallet_state)

  def post_wallet(self):
    # Create a *new* wallet state, indicating that it was last updated by this
    # device, with the updated sequence, and include our local encrypted wallet changes.
    # Don't set self.wallet_state to this until we know that it's accepted by
    # the server.
    if not self.wallet_state:
      print ("No wallet state to post.")
      return

    hmac_key = derive_hmac_key(self.root_password)

    submitted_wallet_state = WalletState(
      encrypted_wallet=self.cur_encrypted_wallet(),
      sequence=self.wallet_state.sequence + 1
    )
    wallet_request = {
      'version': CURRENT_VERSION,
      'token': self.auth_token,
      "encryptedWallet": submitted_wallet_state.encrypted_wallet,
      "sequence": submitted_wallet_state.sequence,
      "hmac": create_hmac(submitted_wallet_state, hmac_key),
    }

    response = requests.post(WALLET_URL, json.dumps(wallet_request))

    if response.status_code == 200:
      # TODO check response version on client side now
      # Our local changes are no longer local, so we reset them
      self._encrypted_wallet_local_changes = ''
      print ('Successfully updated wallet state on server')
    elif response.status_code == 409:
      print ('Wallet state out of date. Getting updated wallet state. Try posting again after this.')
      # Don't return yet! We got the updated state here, so we still process it below.
    else:
      print ('Error', response.status_code)
      print (response.content)
      return

    # Now we get a new wallet back as a response
    # TODO - factor this code into the same thing as the get_wallet function

    new_wallet_state = WalletState(
      encrypted_wallet=response.json()['encryptedWallet'],
      sequence=response.json()['sequence'],
    )
    hmac = response.json()['hmac']
    if not check_hmac(new_wallet_state, hmac_key, hmac):
      print ('Error - bad hmac on new wallet')
      print (response.content)
      return

    if submitted_wallet_state != new_wallet_state and not self._validate_new_wallet_state(new_wallet_state):
      print ('Error - new wallet does not validate')
      print (response.content)
      return

    self.wallet_state = new_wallet_state

    print ("Got new walletState:")
    pprint(self.wallet_state)

  def change_encrypted_wallet(self):
    if not self.wallet_state:
      print ("No wallet state, so we can't add to it yet.")
      return

    self._encrypted_wallet_local_changes += ':' + ''.join(random.choice(string.hexdigits) for x in range(4))

  def cur_encrypted_wallet(self):
    if not self.wallet_state:
      print ("No wallet state, so no encrypted wallet.")
      return

    # The local changes on top of whatever came from the server
    # If we pull new changes from server, we "rebase" these on top of it
    # If we push changes, the full "rebased" version gets committed to the server
    return self.wallet_state.encrypted_wallet + self._encrypted_wallet_local_changes
