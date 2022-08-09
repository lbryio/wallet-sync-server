# Test Client

A couple example flows so it's clear how it works. We're assuming that we're starting with a fresh DB on the server, and that we've created two wallets on the SDK: `"test_wallet_1"` and `"test_wallet_2"`.

## Initial setup and account recovery

Set up a client for each wallet, but with the same sync account (which won't exist on the server yet). This will simulate clients on two different computers.

For this example we will be working with a locally running server so that we don't care about the data. If you want to communicate with `dev.lbry.id`, simply omit the `local=True`.

```
>>> from test_client import Client
>>> c1 = Client("joe2@example.com", "123abc2", 'test_wallet_1', local=True)
Connecting to Wallet API at http://localhost:8090
>>> c2 = Client("joe2@example.com", "123abc2", 'test_wallet_2', local=True)
Connecting to Wallet API at http://localhost:8090
```

Register the account on the server with one of the clients. See the salt seed it generated and sent to the server along with registration.

```
>>> c1.register()
Generating keys...
Done generating keys
Registered
>>> c1.salt_seed
'1d52635c14b34f0fefcf86368d4e0b82e3555de9d3c93a6f22cd5500fd120c0d'
```

Set up the other client. See that it got the same salt seed from the server in the process, which it needs to make sure we have the correct encryption key and login password.

```
>>> c2.update_derived_secrets()
Generating keys...
Done generating keys
>>> c2.salt_seed
'1d52635c14b34f0fefcf86368d4e0b82e3555de9d3c93a6f22cd5500fd120c0d'
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  e52f6e893fe3fa92d677d85f32e77357d68afd313c303a91d3af176ec684aa0d
>>> c2.get_auth_token()
Got auth token:  b9fc2620990447d5f0305ecafc9f75e2a5f928a31bd86806aa8989567cad57d0
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6XBEQgEACPvxgUFW3MGnY9tG5VYh/Hx7iNG6DAX+q4zTbVZM17OQ/5D1+IOjxS7jxOB+dZmtxmo6qwGtizjc4+YBhNk/eKb+uIU8T6HQ4T3m+PiWpedLnBwF4RStPPBp1M2WNFTIZQPKirETPO3GqRQSzveB17A3iESqYTqHnGeE=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6XBEQgEACPvxgUFW3MGnY9tG5VYh/Hx7iNG6DAX+q4zTbVZM17OQ/5D1+IOjxS7jxOB+dZmtxmo6qwGtizjc4+YBhNk/eKb+uIU8T6HQ4T3m+PiWpedLnBwF4RStPPBp1M2WNFTIZQPKirETPO3GqRQSzveB17A3iESqYTqHnGeE=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6gL9aGNjy4U+6mBQZRzx+GS+/1dhl54+5sBzVtBQz51az7HQ3HFI2PjUL7XkeTcjdsaPEKh3eFTQwly9fNFKJIya5YvmtY8zhxe8FCqCkTITrn2EPwZFYXF6E3Wi1gLaPMpZlb2EXIZ1E7Gbg1Uxcpj+s1CB4ttjIZdnFwUrfAw4=')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6gL9aGNjy4U+6mBQZRzx+GS+/1dhl54+5sBzVtBQz51az7HQ3HFI2PjUL7XkeTcjdsaPEKh3eFTQwly9fNFKJIya5YvmtY8zhxe8FCqCkTITrn2EPwZFYXF6E3Wi1gLaPMpZlb2EXIZ1E7Gbg1Uxcpj+s1CB4ttjIZdnFwUrfAw4=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6JwVggGDjqoLy9YUqFXzIltph5bvO46SwJoAbLydlLg1mjfoXksGm9NsWbmYYmBoiXmiIbJPIsj8xfOjO5JlCH+EHSdyjCXizzwClYwgM4UD1+/ltuv1TH7H59cXd6Kztefn4y9IL/97rs+2DxDHM6cb/AdYGohIc3VaCmYBSbYRQFjTbQHaaScW6ntYuXAyE')
'Success'
>>> c2.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6JwVggGDjqoLy9YUqFXzIltph5bvO46SwJoAbLydlLg1mjfoXksGm9NsWbmYYmBoiXmiIbJPIsj8xfOjO5JlCH+EHSdyjCXizzwClYwgM4UD1+/ltuv1TH7H59cXd6Kztefn4y9IL/97rs+2DxDHM6cb/AdYGohIc3VaCmYBSbYRQFjTbQHaaScW6ntYuXAyE')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6xMKOvjQ9RBAWCac5Cj5d30YSI4PaMh3T+99fLdHKJC2RCcwrbhCurNIDBln6QJWCfa3gRp2/sY9k47XwZNsknCTrdIe4c3YJejvL/WCZTzoJ81m9QGbP/05DHQUV5c7z30taIESp4qOFwpSwYMB972gn6ZXOhn1iNDKSCLN3nSLHFnA0arjCAPQof//lJriz')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Merging local changes with remote changes to create latest walletState.
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6xMKOvjQ9RBAWCac5Cj5d30YSI4PaMh3T+99fLdHKJC2RCcwrbhCurNIDBln6QJWCfa3gRp2/sY9k47XwZNsknCTrdIe4c3YJejvL/WCZTzoJ81m9QGbP/05DHQUV5c7z30taIESp4qOFwpSwYMB972gn6ZXOhn1iNDKSCLN3nSLHFnA0arjCAPQof//lJriz')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6L+PCpF1qh1ayai/fqnc7kwa2eBJc1n6L6FuaLps8gdZhY9UdaBMc/BckvgUF9OXR7yOvndrFy73+5EzWxpmffBfZGqq42XjtbmHGScEERjuzra8UB2vLn+N2oe5s+e2O+7lJxPKYBD2pX4xKm3HjKqAso+D0MsWHMz9hqRLFekJfv5pVglUVkweW+h8yNxn1')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6L+PCpF1qh1ayai/fqnc7kwa2eBJc1n6L6FuaLps8gdZhY9UdaBMc/BckvgUF9OXR7yOvndrFy73+5EzWxpmffBfZGqq42XjtbmHGScEERjuzra8UB2vLn+N2oe5s+e2O+7lJxPKYBD2pX4xKm3HjKqAso+D0MsWHMz9hqRLFekJfv5pVglUVkweW+h8yNxn1')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6HieQoVznUMTBF6x643Mg/AQUZadaikkiuRZsw3IaQsapK56WL3IBGrlemOjSH6uTfBWsWaLDMXEz+X7j5wqchSAt/wle2+I9dKgyDdFhWMOaEd61pT6r+lS8O8AbSKUJ6r5FSDgJRE/vz5l4xP/W9AVrK4l0u9ZqpvsKAet3UlfVV48cOnhwgPqlPoGBQ1xF')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6HieQoVznUMTBF6x643Mg/AQUZadaikkiuRZsw3IaQsapK56WL3IBGrlemOjSH6uTfBWsWaLDMXEz+X7j5wqchSAt/wle2+I9dKgyDdFhWMOaEd61pT6r+lS8O8AbSKUJ6r5FSDgJRE/vz5l4xP/W9AVrK4l0u9ZqpvsKAet3UlfVV48cOnhwgPqlPoGBQ1xF')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6ypuX2e/wjiVZZVrLQEwoEuHZH7xhs6B3/awzxH/5WZITlKOo7TvV2Mjke/MSdTk2/YyWhfN8U0e4IwGxKW9VIpnF3ElEtEZxvJBklzDXeDNh5pWMgeZkBH5EempDQ6VzT0206z89EeiCK+3QSofUv7Ob90xNVUOdJq5/OBrG4LAGFh2ZVrh5KnqDm1+d8/ls')
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
WalletState(sequence=8, encrypted_wallet='czo4MTkyOjE2OjE6Kd/DnozNDXYia8yYqrVI6OJ56tDAo5X4/Il+Ein/E6GRQ6K8/niK8Sjx1Cmpf7ecru14QS51pTwlFpS9mbwNE7CZ1wjAZHoLlL5B+dAECkSCFBHgBvq/29cXt6gG7KP+TLRLxZzGtgQRQiq6fsMBIIirw1ZCmpUNQP/PCHIJRfjJS0MNAGN8+srlPv+eUXIn')
'Success'
```

We generate a new salt seed when we change the password

```
>>> c1.salt_seed
'155b6e8a9a8c9406844b6b0c4a40c3204ab1f06668470faa89e28aa89fefe3cf'
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
Got auth token:  68a3db244e21709429e69e67352d02a3b26542c5ef2ac3377e19b17de71942d6
>>> c2.get_auth_token()
Error 401
b'{"error":"Unauthorized: No match for email and/or password"}\n'
Failed to get the auth token. Do you need to update this client's password (set_local_password())?
Or, in the off-chance the user changed their password back and forth, try updating secrets (update_derived_secrets()) to get the latest salt seed.
>>> c2.set_local_password("eggsandwich")
Generating keys...
Done generating keys
>>> c2.salt_seed
'155b6e8a9a8c9406844b6b0c4a40c3204ab1f06668470faa89e28aa89fefe3cf'
>>> c2.get_auth_token()
Got auth token:  3917215675c5cc7fb5c5e24d583fddcd0a14c4370140e2274cf4c5da7eaae7bb
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
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=9, encrypted_wallet='czo4MTkyOjE2OjE6Hspn+wbfHEzSv+1zsM/sFUaJJZuLLP7jLtCl3Ou3OQhXGEpkC0pP7WcbdGdQ+4foakTaB/y/b9All85rJ1ZiGWFnaK8SS9Rd7JT1UCEHs0BhN5+SfIK58yukIefzP39ZlSGUomE3eifOqso8C/gY2FltO96TS8WXx6czxqm6M/dvLk6q10LpODCQEH5auTA6')
'Success'
>>> c1.change_password("starboard")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=10, encrypted_wallet='czo4MTkyOjE2OjE6Cnditb9t+rU56hfcMq6gW+lx1ek3TzyBZ4633FoiWCzTxIenbMyapolU0gnpWHasP8olOoL56LfSGVzP8eKG4JoRsU9VmOYXjkpY9QZCcKomVC4fJ17jPq/e2gJWDSv03pA1xbDhRpXRnZr3wd+37znTUyLpYzRDRAHpb2IGDi9FforobQRNcZUhx0DY8WIR')
'Success'
```
