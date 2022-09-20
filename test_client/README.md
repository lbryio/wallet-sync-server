# Test Client

A couple example flows so it's clear how it works. We're assuming that we're starting with a fresh DB on the server, and that we've created two wallets on the SDK: `"test_wallet_1"` and `"test_wallet_2"`.

## Initial setup and account recovery

Set up a client for each wallet, but with the same sync account (which won't exist on the server yet). This will simulate clients on two different computers.

For this example we will be working with a locally running server so that we don't care about the data. If you want to communicate with `dev.lbry.id`, simply omit the `local=True`.

```
>>> from test_client import Client
>>> import time
>>> email = "joe-%s@example.com" % int(time.time())
>>> c1 = Client("c1", email, "123abc2", 'test_wallet_1', local=True)
Connecting to Wallet API at http://localhost:8090
>>> c2 = Client("c2", email, "123abc2", 'test_wallet_2', local=True)
Connecting to Wallet API at http://localhost:8090
```

Register the account on the server with one of the clients. See the salt seed it generated and sent to the server along with registration.

```
>>> c1.register()
Generating keys...
Done generating keys
Registered
>>> c1.salt_seed
'8a77dcb8b2854c2fecabbde74a721fde5e326164f2cf1a7f6810d0e1f340d043'
```

Set up the other client. See that it got the same salt seed from the server in the process, which it needs to make sure we have the correct encryption key and login password.

```
>>> c2.update_derived_secrets()
Generating keys...
Done generating keys
>>> c2.salt_seed
'8a77dcb8b2854c2fecabbde74a721fde5e326164f2cf1a7f6810d0e1f340d043'
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  9cfbed8d587440b899beb0ea534caaff96981d1f212d83d606642a900deddd1c
>>> c2.get_auth_token()
Got auth token:  99725c84039e323a880e936d14789d8a79b2fc7efdcae08ed4282e05160f5204
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6MArKm6fT20MSDlsRPAxl5gk49wHPwBxjNnouMGsTEi2uQaMwVOyDRETIRLBTPHFHn6Uz5j+9a5o6RfAbChvToaFRpe4FWZGtlSBiRqdnatxnYzwTRK9OxAttPJdO6BJ0tO1pmn5ipSHfkQdbTT/POTSnElsrnfDU+4AtdoO9kNA=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6MArKm6fT20MSDlsRPAxl5gk49wHPwBxjNnouMGsTEi2uQaMwVOyDRETIRLBTPHFHn6Uz5j+9a5o6RfAbChvToaFRpe4FWZGtlSBiRqdnatxnYzwTRK9OxAttPJdO6BJ0tO1pmn5ipSHfkQdbTT/POTSnElsrnfDU+4AtdoO9kNA=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6RbDcGPPGipR3f++iY2IV4TseRvEuZ18HX/SWzzGrw0qbAlChXgSRUTvAlCV1sGyKJEHhBIlGfC+KOCKEGPaK9fx7BmhhcHvCDmwIlcpJ3VwMtwTjxTJZE9+Q8YLOXjZM1RZhPPiCDqxUzNVPaJm2F1MLSn3tDtX5Duz15ll998Y=')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6RbDcGPPGipR3f++iY2IV4TseRvEuZ18HX/SWzzGrw0qbAlChXgSRUTvAlCV1sGyKJEHhBIlGfC+KOCKEGPaK9fx7BmhhcHvCDmwIlcpJ3VwMtwTjxTJZE9+Q8YLOXjZM1RZhPPiCDqxUzNVPaJm2F1MLSn3tDtX5Duz15ll998Y=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6H0sr9zU/SYL2/0abnfmb4y1WqSnRFuylbket0kahuBi42l3RzZunVY5qp7DHFheQ5RNI/KvaEMV6efC9a7EZc/J5nqZOolgdv0dCSPpgwDS0TxUtsCSH6DGZ3htLxqU2r3ZqKX5XCP4f93lTc8loPGvB+e8k6+CYAeXnkS57ske5U6ZYvJtlMMQpYPSVU3xN')
'Success'
>>> c2.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6H0sr9zU/SYL2/0abnfmb4y1WqSnRFuylbket0kahuBi42l3RzZunVY5qp7DHFheQ5RNI/KvaEMV6efC9a7EZc/J5nqZOolgdv0dCSPpgwDS0TxUtsCSH6DGZ3htLxqU2r3ZqKX5XCP4f93lTc8loPGvB+e8k6+CYAeXnkS57ske5U6ZYvJtlMMQpYPSVU3xN')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6W9I5lyVJgKYlT10doJs9NOGHygtBMXyyXVtDI3doUx/eoPwX+qPxHl/Cz3mjDCEWojZgFZqS70ZPBjbRITWQ9iizPWyUG+FtddUVgadW+nCGiSeKHqVmu5n0MihrvNrtKgEka10dmdtj3U4JJsF/0CwlsyzRKLhPgjlJvzn3miW5DOKrNNtJaFmWFJXIJke5')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Merging local changes with remote changes to create latest walletState.
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6W9I5lyVJgKYlT10doJs9NOGHygtBMXyyXVtDI3doUx/eoPwX+qPxHl/Cz3mjDCEWojZgFZqS70ZPBjbRITWQ9iizPWyUG+FtddUVgadW+nCGiSeKHqVmu5n0MihrvNrtKgEka10dmdtj3U4JJsF/0CwlsyzRKLhPgjlJvzn3miW5DOKrNNtJaFmWFJXIJke5')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE60UtNFnnuzYfjdEV2VH/QWO6WTqfSlu0KpUxOm2CuEij9erCmNkiuAQCzihsAZPVSFvt3N9UJRGQdeDRwjN9P2yKr89ED/qBhNHSZzEI8dwR7qrPyqPE5vchiw0UclZPeUdrQiAyyCvZSkThiQNEnwvQyeoucxCZ8P3Gi48Vht48MQ8W07zloMXmndiF81h7G')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE60UtNFnnuzYfjdEV2VH/QWO6WTqfSlu0KpUxOm2CuEij9erCmNkiuAQCzihsAZPVSFvt3N9UJRGQdeDRwjN9P2yKr89ED/qBhNHSZzEI8dwR7qrPyqPE5vchiw0UclZPeUdrQiAyyCvZSkThiQNEnwvQyeoucxCZ8P3Gi48Vht48MQ8W07zloMXmndiF81h7G')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6e2cIHgNvUiveemLTBYx1BDyueh5JNh4ojvIMnSaass14+Li5eKjVCZaU1LQ1gT4zB6ibqoSu3P60MuOc6/A8GUAwh4KVzQfBBJHjmHWN5ZYoBlJY7AdflrFo0mkUwD1pzTYA0+9iexnI+s0v7ya/rFvw77GtErotgLCnlvOoZiJs3EUyRkltjukz2UCy5LKc')
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
Merging local changes with remote changes to create latest walletState.
Got latest walletState:
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6e2cIHgNvUiveemLTBYx1BDyueh5JNh4ojvIMnSaass14+Li5eKjVCZaU1LQ1gT4zB6ibqoSu3P60MuOc6/A8GUAwh4KVzQfBBJHjmHWN5ZYoBlJY7AdflrFo0mkUwD1pzTYA0+9iexnI+s0v7ya/rFvw77GtErotgLCnlvOoZiJs3EUyRkltjukz2UCy5LKc')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6mAo3n7J3aNpyiOLHPaHvOypgop4C2Re+fGzMXrdbSLnrMPxVZxlGS3KRzN58jrQ8gkMUqg3aPviXXyII/W8I8MB3lvRoZPRvD1Mkw48sMhPZ5/QsLvNlJT/29ZzqCo4PQAM3PSVo/ho6i7Bilh7Z6pcm7b8u/fuegbsR5NNdTkQ2nbnZl2sDKH6G3idHGc11')
'Success'
```

# Changing Password

Changing the root password leads to generating a new lbry.id login password, sync password, and hmac key. To avoid complicated scenarios from partial updates, we will account for all three changes on the server by submitting a new password, wallet and hmac in one request (and the server, in turn, will commit all of the changes in one database transaction).

This implies that the client needs to have its local wallet updated before updating their password, just like for a normal wallet update, to keep the sequence values properly incrementing.

There is one exception: if there is no wallet yet saved on the server, the client should not submit a wallet to the server. It should omit the wallet-related fields in the request. (This is for situations where the user is just getting their account set up and needs to change their password. They should not be forced to create and sync a wallet first.). However, at this point in this example, we have a wallet saved so we will submit an update.

```
>>> c1.change_password("eggsandwich")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=8, encrypted_wallet='czo4MTkyOjE2OjE6BuNkOY60NKVP4vIjXKfiy4qV75pDzf3YKzBQwU02yhlUR/Jh6ZZTdpEKrKnScwYTVFrSCO+0V7TyPTEVZrh4eLIHJoLgPgDPGl7BNP1aXH4VH+eroqwPQPMLeMJztInWFJt4U2gM+TqExfjG4pNTm5CbD5qiTkv9RMPLBvapcLbD3xeVQkpAYhTi0I458Hsn')
'Success'
```

We generate a new salt seed when we change the password

```
>>> c1.salt_seed
'e128e66be3c433b30ba40c72d2a42dac8ee84d37a182d4fa2022d4b857d156b0'
```

This operation invalidates all of the user's auth tokens. This prevents other clients from accidentally pushing a wallet encrypted with the old password.

```
>>> c1.get_remote_wallet()
Error 401
b'{"error":"Unauthorized: Token Not Found"}\n'
'Failed to get remote wallet'
>>> c2.get_remote_wallet()
Error 401
b'{"error":"Unauthorized: Token Not Found"}\n'
'Failed to get remote wallet'
```

The client that changed its password can easily get a new token because it has the new password saved locally. The other client needs to update its local password first, which again includes getting the updated salt seed from the server.

```
>>> c1.get_auth_token()
Got auth token:  10c06893d0f6b5c6506d75c55f6bdea361df1514ad8e9d04b19e7cf6852f6352
>>> c2.get_auth_token()
Error 401
b'{"error":"Unauthorized: No match for email and/or password"}\n'
Failed to get the auth token. Do you need to verify your email address? Or update this client's password (set_local_password())?
Or, in the off-chance the user changed their password back and forth, try updating secrets (update_derived_secrets()) to get the latest salt seed.
>>> c2.set_local_password("eggsandwich")
Generating keys...
Done generating keys
>>> c2.salt_seed
'e128e66be3c433b30ba40c72d2a42dac8ee84d37a182d4fa2022d4b857d156b0'
>>> c2.get_auth_token()
Got auth token:  90689d10cda131747d9dcaddbd474806c16e3d218b54cff4a4ea8fc9d2888e23
```

We don't allow password changes if we have pending wallet changes to push. This is to prevent a situation where the user has to merge local and remote changes in the middle of a password change.

```
>>> c1.set_preference('animal', 'leemur')
{'animal': 'leemur'}
>>> c1.change_password("starboard")
Generating keys...
Done generating keys
Local changes found. Update remote wallet before changing password.
'Failure'
```

If we update the wallet first, we can do it.

```
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=9, encrypted_wallet='czo4MTkyOjE2OjE6JB6y6xDMds4pp4eDh8TDfXvpENnfaQztvwua+ipXE7XSlkOJDjQ7GEv7NtnSPP+Whsny1p6GIWgf22WJ76KGMg16oMTxXWVwK0yTzSiQVhQhc5lNHGMSQDu6wOlHhQAfwta3xwMKbFnklq0GlTRYayeUNspFmFRxKulC5dsdU72V6bPnk3lE5K15JKpNGSQ5')
'Success'
>>> c1.change_password("starboard")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=10, encrypted_wallet='czo4MTkyOjE2OjE64YLYwQx95KgCS+kqWhVm3pHBV18bFANe+Kb3Fi3+m9k73p78l/4RF82pxxsYgJeZPrAcpV3pz21dQPVgBDuWszDqk8LkvSQXK/OAmrL3gPFmvuXEV4E504UPcxWeBqiFHNxV1cbWs7UJf31ofVJTEDwqhd0jVitJr/U8+0YU1enEnQd6UavKBxhiaIcy4+xd')
'Success'
>>> c1.get_auth_token()
Got auth token:  ab5820fa0f92ca11698b88ca11c7a38e778e4760b85e2fdaedd761561d87951e
>>> c2.set_local_password("starboard")
Generating keys...
Done generating keys
>>> c2.get_auth_token()
Got auth token:  4c32f3348d51662519b7d8675b69571a1ce4d76de88a7b495a4848c5dbb4da7f
```

# Websockets

A client can make a websocket connection to the server and receive notifications whenever another client updates the wallet on the server. The message will contain the sequence number so that the client can know whether they happen to be up to date. (The client that made the update will of course be up to date).

This test client will have a thread listening to the websocket which just prints info about new messages as they come in. A real client would likely choose to get the latest wallet from the server as soon as a messag ecomes through, assuming the sequence is newer than what the client has.

```
>>> c1.start_websocket()
c1 connected for now
>>> c2.start_websocket()
c2 connected for now
```

Now make an update and see:

```
>>> c1.update_remote_wallet()
c2 got notified of a wallet update, sequence=11. If your client is behind this sequence, you should get the latest from the server.
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=11, encrypted_wallet='czo4MTkyOjE2OjE6h5sx/erPDmRUd7lUwhW9xDkWHmWOvooebx99Je6WMvG+XXd98MjOuBYTIFHbtr0XzR2ARzvNCmnvxSUBGfRoCiF9Um6OqimJTuc636E2pCgLzvEq8W39qYb3enlu6zd+NiGUXEo85j9WY1FvxrSPBxvV21cM4HLWijDGBob/adYTx33sT8o6tZ/18axxNdNK')
'Success'
c1 got notified of a wallet update, sequence=11. If your client is behind this sequence, you should get the latest from the server.
```

Update again and we'll see the new sequence number:

```
>>> c1.update_remote_wallet()
c2 got notified of a wallet update, sequence=12. If your client is behind this sequence, you should get the latest from the server.
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=12, encrypted_wallet='czo4MTkyOjE2OjE6WvomdZhbeg1ypEkHVR07Z4yI2uj1thLkzolgIcvPfE2i+vexeTfG6WE+bsVlhMwn7wXzQ3tIeW7P8IGpAS+HtcySxXyAyZXJxX/2N6fzqd9Dai8pNO1Ed2RyxGGjMf4spMDT3wJOQ93Plsc96y36iPIzNXQ8gtL15IjAfbiImR+KPmIG8E0IDyYVuOoaTF+A')
'Success'
c1 got notified of a wallet update, sequence=12. If your client is behind this sequence, you should get the latest from the server.
```

When we change a password, just as all auth tokens are invalidated, all sockets are also disconnected.

```
>>> c1.change_password("ihatesockets")
Generating keys...
Done generating keys
c2 disconnected for now: code = 1005 (no status code [internal]), no reason
c1 disconnected for now: code = 1005 (no status code [internal]), no reason
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=13, encrypted_wallet='czo4MTkyOjE2OjE6GGAs7vW0bt+fzo0xNwQQLd0nY9CM9r1flMtwXNNktXuEA0TkaWVSpP1r/pyN3qLkY8it+OAD2dm44usxzEtNnq1hH40vDPgAYpMtSXRVV1aPTZK6Zv2W1MQN88N4qI592tqpGEm8q2FtqrvA61Rf/CIUGbVKmf5jiPx9aiON6t3etENkZD9t+DEMJo6TOlO9')
'Success'
```
