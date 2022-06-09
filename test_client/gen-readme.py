# Generate the README since I want real behavior interspersed with comments
# Come to think of it, this is accidentally a pretty okay integration test for client and server
# NOTE - delete the database before running this, or else you'll get an error for registering. also we want the wallet to start empty

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

A couple example flows so it's clear how it works.
""")

print("""## Initial setup and account recovery

Set up two clients with the same account (which won't exist on the server yet).
""")

code_block("""
from test_client import Client
c1 = Client()
c2 = Client()
c1.set_account("joe2@example.com", "123abc2")
c2.set_account("joe2@example.com", "123abc2")
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

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. This increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

Note that after POSTing, it says it "got" a new wallet. This is because the POST endpoint also returns the latest version. The purpose of this will be explained in "Conflicts" below.
""")

code_block("""
c1.new_wallet_state()
c1.post_wallet()
""")

print("""
With the other client, GET it from the server. Note that both clients have the same data now.
""")

code_block("""
c2.get_wallet()
""")

print("""
## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.
""")

code_block("""
c2.post_wallet()
c1.get_wallet()
""")

print("""
## Wallet Changes

For demo purposes, this test client represents each change to the wallet by appending segments separated by `:` so that we can more easily follow the history. (The real app will not actually edit the wallet in the form of an append log.)
""")

code_block("""
c1.cur_encrypted_wallet()
c1.change_encrypted_wallet()
c1.cur_encrypted_wallet()
""")

print("""
The wallet is synced between the clients.
""")

code_block("""
c1.post_wallet()
c2.get_wallet()
c2.cur_encrypted_wallet()
""")

print("""
## Merging Changes

Both clients create changes. They now have diverging wallets.
""")

merge_base = c2.cur_encrypted_wallet()

code_block("""
c1.change_encrypted_wallet()
c2.change_encrypted_wallet()
c1.cur_encrypted_wallet()
c2.cur_encrypted_wallet()
""")

print("""
One client POSTs its change first.
""")

code_block("""
c1.post_wallet()
""")

print("""
The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally.

The _merge base_ that a given client uses is the last version that it successfully got from or POSTed to the server. You can see the merge base here: `"%s"`, the first part of the wallet which both clients had in common before the merge.
""" % merge_base)

code_block("""
c2.get_wallet()
c2.cur_encrypted_wallet()
""")

print("""
Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.
""")

code_block("""
c2.post_wallet()
c1.get_wallet()
c1.cur_encrypted_wallet()
""")

print("""
## Conflicts

A client cannot POST if it is not up to date. It needs to merge in any new changes on the server before POSTing its own changes. For convenience, if a conflicting POST request is made, the server responds with the latest version of the wallet state (just like a GET request). This way the client doesn't need to make a second request to perform the merge.

(If a non-conflicting POST request is made, it responds with the same wallet state that the client just POSTed, as it is now the server's current wallet state)
""")

code_block("""
c2.change_encrypted_wallet()
c2.post_wallet()
c1.change_encrypted_wallet()
c1.post_wallet()
""")

print("""
Now the merge is complete, and the client can make a second POST request containing the merged wallet.
""")

code_block("""
c1.post_wallet()
""")
