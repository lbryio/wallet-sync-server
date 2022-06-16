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
time.sleep(3)

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
Register the account on the server with one of the clients.
""")

code_block("""
c1.register()
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

_Note that after POSTing, it says it "got" a new wallet. This is because the POST endpoint also returns the latest version. The purpose of this will be explained in "Conflicts" below._
""")

code_block("""
c1.init_wallet_state()
c1.update_remote_wallet()
""")

print("""
Now, call `init_wallet_state` with the other client. This time, `init_wallet_state` will GET the wallet from the server. In general, `init_wallet_state` is used to set up a new client; first it checks the server, then failing that, it initializes it locally. (In a real client, it would save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)
""")

code_block("""
c2.init_wallet_state()
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

# Make sure the next preference changes have a later timestamp!
time.sleep(3)

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
We try to POST both of them to the server, but the second one fails because of the conflict. Instead, merges the two locally:
""")

code_block("""
c2.update_remote_wallet()
c1.update_remote_wallet()
c1.get_preferences()
""")

print("""
Now that the merge is complete, the client can make a second POST request containing the merged wallet.
""")

code_block("""
c1.update_remote_wallet()
""")
