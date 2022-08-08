#!/bin/python3

# Generate the README since I want real behavior interspersed with comments
# Come to think of it, this is accidentally a pretty okay integration test for client and server
# NOTE - delete the database before running this, or else you'll get an error for registering. also we want the wallet to start empty
# NOTE - in the SDK, create wallets called "test_wallet_1" and "test_wallet_2"

import time
from test_client import LBRYSDK
# reset all of the preferences so we can run our example
for wallet in ['test_wallet_1', 'test_wallet_2']:
  for pref_key in ['car', 'animal']:
    LBRYSDK.set_preference(wallet, pref_key, '')

# Make sure the next preference changes have a later timestamp!
time.sleep(1.1)

def code_block(code):
  print ("```")
  for line in code.strip().split('\n'):
    print(">>> " + line)
    if ' = ' in line or "import" in line:
      exec('global c1, c2\n' + line)
    else:
      result = eval(line)
      if result is not None:
        print(repr(result))
      if 'set_preference' in line:
        # Make sure the next preference changes have a later timestamp!
        time.sleep(1.1)
  print ("```")

print("""# Test Client

A couple example flows so it's clear how it works. We're assuming that we're starting with a fresh DB on the server, and that we've created two wallets on the SDK: `"test_wallet_1"` and `"test_wallet_2"`.
""")

print("""## Initial setup and account recovery

Set up a client for each wallet, but with the same sync account (which won't exist on the server yet). This will simulate clients on two different computers.

For this example we will be working with a locally running server so that we don't care about the data. If you want to communicate with `dev.lbry.id`, simply omit the `local=True`.
""")

code_block("""
from test_client import Client
c1 = Client("joe2@example.com", "123abc2", 'test_wallet_1', local=True)
c2 = Client("joe2@example.com", "123abc2", 'test_wallet_2', local=True)
""")

print("""
Register the account on the server with one of the clients. See the salt seed it generated and sent to the server along with registration.
""")

code_block("""
c1.register()
c1.salt_seed
""")

print("""
Set up the other client. See that it got the same salt seed from the server in the process, which it needs to make sure we have the correct encryption key and login password.
""")

code_block("""
c2.update_secrets()
c2.salt_seed
""")

print("""
Now that the account exists, grab an auth token with both clients.
""")

code_block("""
c1.get_auth_token()
c2.get_auth_token()
""")

print("""
## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.
""")

code_block("""
c1.init_wallet_state()
c1.update_remote_wallet()
""")

print("""
Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)
""")

code_block("""
c2.init_wallet_state()
c2.get_remote_wallet()
""")

print("""
## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.
""")

code_block("""
c2.update_remote_wallet()
c1.get_remote_wallet()
""")

print("""
## Wallet Changes

We'll track changes to the wallet by changing and looking at preferences in the locally saved wallet. We see that both clients have settings blank. We change a preference on one client:

""")

code_block("""
c1.get_preferences()
c2.get_preferences()
c1.set_preference('animal', 'cow')
c1.get_preferences()
""")

print("""
The wallet is synced between the clients. The client with the changed preference sends its wallet to the server, and the other one GETs it locally.
""")

code_block("""
c1.update_remote_wallet()
c2.get_remote_wallet()
c2.get_preferences()
""")

print("""
## Merging Changes

Both clients create changes. They now have diverging wallets.
""")

code_block("""
c1.set_preference('car', 'Audi')
c2.set_preference('animal', 'horse')
c1.get_preferences()
c2.get_preferences()
""")

print("""
One client POSTs its change first.
""")

code_block("""
c1.update_remote_wallet()
""")

print("""
The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.
""")

code_block("""
c2.get_remote_wallet()
c2.get_preferences()
""")

print("""
Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.
""")

code_block("""
c2.update_remote_wallet()
c1.get_remote_wallet()
c1.get_preferences()
""")

print("""
Note that we're sidestepping the question of merging different changes to the same preference. The SDK resolves this, again, by timestamps. But ideally we would resolve such an issue with a user interaction (particularly if one of the changes involves _deleting_ the preference altogether). Using timestamps as the SDK does is a holdover from the current system, so we won't distract ourselves by demonstrating it here.

## Conflicts

A client cannot POST if it is not up to date. It needs to merge in any new changes on the server before POSTing its own changes. For convenience, if a conflicting POST request is made, the server responds with the latest version of the wallet state (just like a GET request). This way the client doesn't need to make a second request to perform the merge.

(If a non-conflicting POST request is made, it responds with the same wallet state that the client just POSTed, as it is now the server's current wallet state)

So for example, let's say we create diverging changes in the wallets:
""")

code_block("""
_ = c2.set_preference('animal', 'beaver')
_ = c1.set_preference('car', 'Toyota')
c2.get_preferences()
c1.get_preferences()
""")

print("""
We try to POST both of them to the server. The second one fails because of the conflict, and we see that its preferences don't change yet.
""")

code_block("""
c2.update_remote_wallet()
c1.update_remote_wallet()
c1.get_preferences()
""")

print("""
The client that is out of date will then call `get_remote_wallet`, which GETs and automatically merges in the latest wallet. We see the preferences are now merged. Now it can make a second POST request containing the merged wallet.
""")

code_block("""
c1.get_remote_wallet()
c1.get_preferences()
c1.update_remote_wallet()
""")

print("""
# Changing Password

Changing the root password leads to generating a new lbry.id login password, sync password, and hmac key. To avoid complicated scenarios from partial updates, we will account for all three changes on the server by submitting a new password, wallet and hmac in one request (and the server, in turn, will commit all of the changes in one database transaction).

This implies that the client needs to have its local wallet updated before updating their password, just like for a normal wallet update, to keep the sequence values properly incrementing.

There is one exception: if there is no wallet yet saved on the server, the client should not submit a wallet to the server. It should omit the wallet-related fields in the request. (This is for situations where the user is just getting their account set up and needs to change their password. They should not be forced to create and sync a wallet first.). However, at this point in this example, we have a wallet saved so we will submit an update.
""")

code_block("""
c1.change_password("eggsandwich")
""")

print("""
We generate a new salt seed when we change the password
""")

code_block("""
c1.salt_seed
""")

print("""
This operation invalidates all of the user's auth tokens. This prevents other clients from accidentally pushing a wallet encrypted with the old password.
""")

code_block("""
c1.get_remote_wallet()
c2.get_remote_wallet()
""")

print("""
The client that changed its password can easily get a new token because it has the new password saved locally. The other client needs to update its local password first, which again includes getting the updated salt seed from the server.
""")

code_block("""
c1.get_auth_token()
c2.get_auth_token()
c2.set_local_password("eggsandwich")
c2.salt_seed
c2.get_auth_token()
""")

print("""
We don't allow password changes if we have pending wallet changes to push. This is to prevent a situation where the user has to merge local and remote changes in the middle of a password change.
""")

code_block("""
c1.set_preference('animal', 'leemur')
c1.change_password("starboard")
c1.update_remote_wallet()
c1.change_password("starboard")
""")
