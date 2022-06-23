# Test Client

A couple example flows so it's clear how it works. We're assuming that we're starting with a fresh DB on the server, and that we've created two wallets on the SDK: `"test_wallet_1"` and `"test_wallet_2"`.

## Initial setup and account recovery

Set up a client for each wallet, but with the same sync account (which won't exist on the server yet). This will simulate clients on two different computers.

For this example we will be working with a locally running server so that we don't care about the data. If you want to communicate with `dev.lbry.id`, simply omit the `local=True`.

```
>>> from test_client import Client
>>> c1 = Client("joe2@example.com", "123abc2", 'test_wallet_1', local=True)
>>> c2 = Client("joe2@example.com", "123abc2", 'test_wallet_2', local=True)
```

Register the account on the server with one of the clients.

```
>>> c1.register()
Registered
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  4a3d9b8569c3b06079ff26d60ebc56db6254305217602c19b0af6e02db6d95d7
>>> c2.get_auth_token()
Got auth token:  33fd77031ccaec966018867e960446bf39d51a3c492c3d997d5f1aa13c75298d
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6/MNVSMrjIqPzrD/oaub++J3lc5qW+baxD0EI6n5/XqGgRsUND3G7fqRsn/riULM4zap+jI8XgW6l1rieJWGZXPQvIZJP8B7gQvBDfzlY0BxUgECeX38I5EtRFNWU3sTwmAaAaDuBpaBXvnf2hu4SEp5xl/OQVg9h+BluTZBdLSU=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6/MNVSMrjIqPzrD/oaub++J3lc5qW+baxD0EI6n5/XqGgRsUND3G7fqRsn/riULM4zap+jI8XgW6l1rieJWGZXPQvIZJP8B7gQvBDfzlY0BxUgECeX38I5EtRFNWU3sTwmAaAaDuBpaBXvnf2hu4SEp5xl/OQVg9h+BluTZBdLSU=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6MIPxgbxNGbaZWboH6ci6wBT3izdpb/B3JYdl3nJdQn6EV54W4QaYUvuUxMa5XngiXlNLcLbmFRqeYj/mgAbEVXRKLyLQxjB7rIhGcRxsHbzGR8YDMVvP+m5dWaxevlZc7cEZkpRQKfFyuc+pnjPEk9SUvEgioN1Hxir6DonMqlA=')
'Success'
>>> c1.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6MIPxgbxNGbaZWboH6ci6wBT3izdpb/B3JYdl3nJdQn6EV54W4QaYUvuUxMa5XngiXlNLcLbmFRqeYj/mgAbEVXRKLyLQxjB7rIhGcRxsHbzGR8YDMVvP+m5dWaxevlZc7cEZkpRQKfFyuc+pnjPEk9SUvEgioN1Hxir6DonMqlA=')
'Success'
```

## Wallet Changes

We'll track changes to the wallet by changing and looking at preferences in the locally saved wallet. We see that both clients have settings blank. We change a preference on one client:


```
>>> c1.get_preferences()
{'animal': '', 'car': ''}
>>> c2.get_preferences()
{'animal': '', 'car': ''}
>>> c1.set_preference('animal', 'cow')
{'animal': 'cow'}
>>> c1.get_preferences()
{'animal': 'cow', 'car': ''}
```

The wallet is synced between the clients. The client with the changed preference sends its wallet to the server, and the other one GETs it locally.

```
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6YUEKfjxhUXeHrNbPuWpMt5o/6H5fSSKFZAMkb8YugMGEHzVAZDfGMdowwdycXkyTZtPRiMSs+kgOX8BLomcz/I+de8b1EsXribYR05sgySRJiPoW8VBRlmgbRapZ9iGaxvgJJWmVAO42beNWtnuE3bdpDtWtZjgcXWq6lnhNlETmKEEPthezGB8svHPHt/rJ')
'Success'
>>> c2.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6YUEKfjxhUXeHrNbPuWpMt5o/6H5fSSKFZAMkb8YugMGEHzVAZDfGMdowwdycXkyTZtPRiMSs+kgOX8BLomcz/I+de8b1EsXribYR05sgySRJiPoW8VBRlmgbRapZ9iGaxvgJJWmVAO42beNWtnuE3bdpDtWtZjgcXWq6lnhNlETmKEEPthezGB8svHPHt/rJ')
'Success'
>>> c2.get_preferences()
{'animal': 'cow', 'car': ''}
```

## Merging Changes

Both clients create changes. They now have diverging wallets.

```
>>> c1.set_preference('car', 'Audi')
{'car': 'Audi'}
>>> c2.set_preference('animal', 'horse')
{'animal': 'horse'}
>>> c1.get_preferences()
{'animal': 'cow', 'car': 'Audi'}
>>> c2.get_preferences()
{'animal': 'horse', 'car': ''}
```

One client POSTs its change first.

```
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6ZAO02VSfc0UTNcKJosuTzdpB1GCRw+f1bCrR/1aFDGoK5Iq/OyKXygp3p2trj2EU1SUfp6m/FiWYdN920uzpaQnIbOlEs6anPpd3alNQmNfuT1s8bKnliO6so657VjZf0QdadDrCVa8WZMiuHY+wP2H5LpzDIrRYrzNyyUuhffbh8yk8cQhgRScFKczpAnu+')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6ZAO02VSfc0UTNcKJosuTzdpB1GCRw+f1bCrR/1aFDGoK5Iq/OyKXygp3p2trj2EU1SUfp6m/FiWYdN920uzpaQnIbOlEs6anPpd3alNQmNfuT1s8bKnliO6so657VjZf0QdadDrCVa8WZMiuHY+wP2H5LpzDIrRYrzNyyUuhffbh8yk8cQhgRScFKczpAnu+')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6cat6gX80ib+t6bX9QlBw3jspj4jJ6U8AGULRDPNa8PbL4CX6ohZoXkt+duNYPxWDdyl8xqhwisWXTXkuGUBwP2zrVmZC3TNt5A9Pk/y/tNgMz50CY3JmNYcbCeZyoY+uV+cMfdO+n3p3hYriNKgn539NC6ug80U/2heevVax4NgMAF0lWEBM2E886+KkvfHG')
'Success'
>>> c1.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6cat6gX80ib+t6bX9QlBw3jspj4jJ6U8AGULRDPNa8PbL4CX6ohZoXkt+duNYPxWDdyl8xqhwisWXTXkuGUBwP2zrVmZC3TNt5A9Pk/y/tNgMz50CY3JmNYcbCeZyoY+uV+cMfdO+n3p3hYriNKgn539NC6ug80U/2heevVax4NgMAF0lWEBM2E886+KkvfHG')
'Success'
>>> c1.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Note that we're sidestepping the question of merging different changes to the same preference. The SDK resolves this, again, by timestamps. But ideally we would resolve such an issue with a user interaction (particularly if one of the changes involves _deleting_ the preference altogether). Using timestamps as the SDK does is a holdover from the current system, so we won't distract ourselves by demonstrating it here.

## Conflicts

A client cannot POST if it is not up to date. It needs to merge in any new changes on the server before POSTing its own changes. For convenience, if a conflicting POST request is made, the server responds with the latest version of the wallet state (just like a GET request). This way the client doesn't need to make a second request to perform the merge.

(If a non-conflicting POST request is made, it responds with the same wallet state that the client just POSTed, as it is now the server's current wallet state)

So for example, let's say we create diverging changes in the wallets:

```
>>> _ = c2.set_preference('animal', 'beaver')
>>> _ = c1.set_preference('car', 'Toyota')
>>> c2.get_preferences()
{'animal': 'beaver', 'car': 'Audi'}
>>> c1.get_preferences()
{'animal': 'horse', 'car': 'Toyota'}
```

We try to POST both of them to the server. The second one fails because of the conflict, and we see that its preferences don't change yet.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6IQ+uyjKiGAIEjoNliOsANoq2h/exQpwordUQFVbbHVhj27UbJS7ykMV4or5avEwNo+aCYC8j7HEqqaPnhvNYeeyPbmpfZS0lU7MXBehoqvIPR3GyTLM002t7SUrB+KxdvUX8RAamjiahDI8OeTOBmYhgQLSZt/ZDtRL/3f5l1JgLCjEbVKJY6Pim0hk7AlpK')
'Success'
>>> c1.update_remote_wallet()
Submitted wallet is out of date.
Could not update. Need to get new wallet and merge
'Failure'
>>> c1.get_preferences()
{'animal': 'horse', 'car': 'Toyota'}
```

The client that is out of date will then call `get_remote_wallet`, which GETs and automatically merges in the latest wallet. We see the preferences are now merged. Now it can make a second POST request containing the merged wallet.

```
>>> c1.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6IQ+uyjKiGAIEjoNliOsANoq2h/exQpwordUQFVbbHVhj27UbJS7ykMV4or5avEwNo+aCYC8j7HEqqaPnhvNYeeyPbmpfZS0lU7MXBehoqvIPR3GyTLM002t7SUrB+KxdvUX8RAamjiahDI8OeTOBmYhgQLSZt/ZDtRL/3f5l1JgLCjEbVKJY6Pim0hk7AlpK')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE63OwBCfczOA+n0EMe0lHPwVvmrXsJwKJXGPYFSmdDseHbd3HRpOZ/Id5WeOuata5/dHJ4vdaaw8RNfpgR4KVzOkM5BUZNxzBaVf/BEYL8nJcbv7l5ZLs6Q15IqvlmZ3HBPVzxO/WYqm4aL9+CNeoYG2LzaIxsnzf31ZoG9I78B6wxK5JXCjDS+nuh/4NM+REE')
'Success'
```
