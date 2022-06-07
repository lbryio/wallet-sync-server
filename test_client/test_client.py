#!/bin/python3
import random, string, json, uuid, requests, hashlib
from pprint import pprint

BASE_URL = 'http://localhost:8090'
AUTH_URL = BASE_URL + '/auth/full'
REGISTER_URL = BASE_URL + '/signup'
WALLET_STATE_URL = BASE_URL + '/wallet-state'

def wallet_state_sequence(wallet_state):
  if 'deviceId' not in wallet_state:
    return 0
  return wallet_state['lastSynced'][wallet_state['deviceId']]

# TODO - do this correctly
def create_login_password(root_password):
    return hashlib.sha256(root_password.encode('utf-8')).hexdigest()[:32]

# TODO - do this correctly
def create_encryption_key(root_password):
    return hashlib.sha256(root_password.encode('utf-8')).hexdigest()[32:]

# TODO - do this correctly
def check_hmac(wallet_state, encryption_key, hmac):
    return hmac == 'Good HMAC'

# TODO - do this correctly
def create_hmac(wallet_state, encryption_key):
    return 'Good HMAC'

class Client():
  def _validate_new_wallet_state(self, new_wallet_state):
    if self.wallet_state is None:
      # All of the validations here are in reference to what the device already
      # has. If this device is getting a wallet state for the first time, there
      # is no basis for comparison.
      return True

    # Make sure that the new sequence is overall later.
    if wallet_state_sequence(new_wallet_state) <= wallet_state_sequence(self.wallet_state):
      return False

    for dev_id in self.wallet_state['lastSynced']:
      if dev_id == self.device_id:
        # Check if the new wallet has the latest changes from this device
        if new_wallet_state['lastSynced'][dev_id] != self.wallet_state['lastSynced'][dev_id]:
          return False
      else:
        # Check if the new wallet somehow regressed on any of the other devices
        # This most likely means a bug in another client
        if new_wallet_state['lastSynced'][dev_id] < self.wallet_state['lastSynced'][dev_id]:
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
    self.wallet_state = {'lastSynced': {}, 'encryptedWallet': ''}

    # TODO - actual encryption with encryption_key
    self._encrypted_wallet_local_changes = ''

  def set_account(self, email, root_password):
    self.email = email
    self.root_password = root_password

  def register(self):
    body = json.dumps({
      'email': self.email,
      'password': create_login_password(self.root_password),
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
      'password': create_login_password(self.root_password),
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
  def get_wallet_state(self):
    params = {
      'token': self.auth_token,
    }
    response = requests.get(WALLET_STATE_URL, params=params)
    if response.status_code != 200:
      print ('Error', response.status_code)
      print (response.content)
      return

    new_wallet_state_str = json.loads(response.content)['walletStateJson']
    new_wallet_state = json.loads(new_wallet_state_str)
    encryption_key = create_encryption_key(self.root_password)
    hmac = json.loads(response.content)['hmac']
    if not check_hmac(new_wallet_state_str, encryption_key, hmac):
      print ('Error - bad hmac on new wallet')
      print (response.content)
      return

    # In reality, we'd examine, merge, verify, validate etc this new wallet state.
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

  def post_wallet_state(self):
    # Create a *new* wallet state, indicating that it was last updated by this
    # device, with the updated sequence, and include our local encrypted wallet changes.
    # Don't set self.wallet_state to this until we know that it's accepted by
    # the server.
    if not self.wallet_state:
      print ("No wallet state to post.")
      return

    submitted_wallet_state = {
      "deviceId": self.device_id,
      "lastSynced": dict(self.wallet_state['lastSynced']),
      "encryptedWallet": self.cur_encrypted_wallet(),
    }
    submitted_wallet_state['lastSynced'][self.device_id] = wallet_state_sequence(self.wallet_state) + 1

    encryption_key = create_encryption_key(self.root_password)

    submitted_wallet_state_str = json.dumps(submitted_wallet_state)
    submitted_wallet_state_hmac = create_hmac(submitted_wallet_state_str, encryption_key)
    body = json.dumps({
      'token': self.auth_token,
      'walletStateJson': submitted_wallet_state_str,
      'hmac': submitted_wallet_state_hmac
    })
    response = requests.post(WALLET_STATE_URL, body)

    if response.status_code == 200:
      # Our local changes are no longer local, so we reset them
      self._encrypted_wallet_local_changes = ''
      print ('Successfully updated wallet state on server')
    elif response.status_code == 409:
      print ('Wallet state out of date. Getting updated wallet state. Try again.')
      # Don't return yet! We got the updated state here, so we still process it below.
    else:
      print ('Error', response.status_code)
      print (response.content)
      return

    # Now we get a new wallet state back as a response
    # TODO - factor this into the same thing as the get_wallet_state function

    new_wallet_state_str = json.loads(response.content)['walletStateJson']
    new_wallet_state_hmac = json.loads(response.content)['hmac']
    new_wallet_state = json.loads(new_wallet_state_str)
    if not check_hmac(new_wallet_state_str, encryption_key, new_wallet_state_hmac):
      print ('Error - bad hmac on new wallet')
      print (response.content)
      return

    # In reality, we'd examine, merge, verify, validate etc this new wallet state.
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
    return self.wallet_state['encryptedWallet'] + self._encrypted_wallet_local_changes
