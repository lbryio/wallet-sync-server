# Test Client

A couple example flows so it's clear how it works. We're assuming that we're starting with a fresh DB on the server, and that we've created two wallets on the SDK: `"test_wallet_1"` and `"test_wallet_2"`.

## Initial setup and account recovery

Set up a client for each wallet, but with the same sync account (which won't exist on the server yet). This will simulate clients on two different computers.

For this example we will be working with a locally running server so that we don't care about the data. If you want to communicate with `dev.lbry.id`, simply omit the `local=True`.

```
>>> from test_client import Client
>>> c1 = Client("joe2@example.com", "123abc2", 'test_wallet_1', local=True)
Generating keys...
Done generating keys
>>> c2 = Client("joe2@example.com", "123abc2", 'test_wallet_2', local=True)
Generating keys...
Done generating keys
```

Register the account on the server with one of the clients.

```
>>> c1.register()
Registered
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  e244aae31bb2070d9269c14706a3a352ddda5e090fefc315bd4d28b8bc787278
>>> c2.get_auth_token()
Got auth token:  c77afe32288a9bf45e55b8fba5aca2b9d3388a277ad89db6fff179337afbd62b
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6+oSc2AE5FY971fW2kQqFvXnen5RD8RU9pMjaKEnvFE8XrdXXogVooiu9Q/099eT8Y9UePoER/aphmzJBb/fwNTOWanFsPCdEObmwfuL1OLPJ+FuAJ07am8TUSJEy12yuMqtQSj6kVF2aMa4oABthKaZ00sx98HkkdUo6sWedY0o=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6+oSc2AE5FY971fW2kQqFvXnen5RD8RU9pMjaKEnvFE8XrdXXogVooiu9Q/099eT8Y9UePoER/aphmzJBb/fwNTOWanFsPCdEObmwfuL1OLPJ+FuAJ07am8TUSJEy12yuMqtQSj6kVF2aMa4oABthKaZ00sx98HkkdUo6sWedY0o=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6mTAOkeMQKuJWirqfyDBjzvneKqcDdNO7UzZ2EdGFs1iW89WMU5UxL//hetnIcXLFFh0SqUjCfj5heyLKEvYY5wJQ0cmIJZEAiPFIZWUjju8J8UEeRl5JWW89x3qhUNrog5a7PnIi/AIRAm6tl7gfzMoujHBWiLPM4xKOO8wX9dw=')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6mTAOkeMQKuJWirqfyDBjzvneKqcDdNO7UzZ2EdGFs1iW89WMU5UxL//hetnIcXLFFh0SqUjCfj5heyLKEvYY5wJQ0cmIJZEAiPFIZWUjju8J8UEeRl5JWW89x3qhUNrog5a7PnIi/AIRAm6tl7gfzMoujHBWiLPM4xKOO8wX9dw=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6uTrpDaroi9aQ0D5rtu8kietZspbFSlyQyEqqfRKA+bMp4Ob7VK3lznxByGs67IpPm2Z0ZorMzaNzkuCghXh/N6YDjQFhZTUWxVo9N10M1bi++2rq2tK4iagARbWPar+Ju8zba2UcknOLZKzphYU1t8EXPykpZUonXO894ljOb2kKEs7eltudGvdRB2DqNgH2')
'Success'
>>> c2.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6uTrpDaroi9aQ0D5rtu8kietZspbFSlyQyEqqfRKA+bMp4Ob7VK3lznxByGs67IpPm2Z0ZorMzaNzkuCghXh/N6YDjQFhZTUWxVo9N10M1bi++2rq2tK4iagARbWPar+Ju8zba2UcknOLZKzphYU1t8EXPykpZUonXO894ljOb2kKEs7eltudGvdRB2DqNgH2')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6QQcktx8tncvrkjGJZk7o37IZ26AsGJnNLif2JiPIZnyRkINakzeU57cryvom9pG0qVdFFdDTAKKIreEj//yJt4pj40rhdsQ8nX6qCuN0nkcHtnpCNcTSmXlRfC/4WDfL5Mq5/HWYVVeQ54GlPp3n2Fj9910TlXVRibp6RO2P98f6cEP8kHM7s+efgLtCRmVK')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Merging local changes with remote changes to create latest walletState.
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6QQcktx8tncvrkjGJZk7o37IZ26AsGJnNLif2JiPIZnyRkINakzeU57cryvom9pG0qVdFFdDTAKKIreEj//yJt4pj40rhdsQ8nX6qCuN0nkcHtnpCNcTSmXlRfC/4WDfL5Mq5/HWYVVeQ54GlPp3n2Fj9910TlXVRibp6RO2P98f6cEP8kHM7s+efgLtCRmVK')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6t9OMFtRl0D4E4YJoE8zR0VuteEroiRyOUgXEhjUBuG0stbwqO/WoNuydNxmRtVMLWgHV5DUlUGZKlTBsuf/fJ6svMdUU7R34uYsSve5ioJw+FBY/w25CYRpa49YZfNhu5YOtmeLHF7AuTMBoc2kkyJj0Jg0IhjqfORIQiifW0YwaWh/eEch9Kzxi+d5DGMaL')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6t9OMFtRl0D4E4YJoE8zR0VuteEroiRyOUgXEhjUBuG0stbwqO/WoNuydNxmRtVMLWgHV5DUlUGZKlTBsuf/fJ6svMdUU7R34uYsSve5ioJw+FBY/w25CYRpa49YZfNhu5YOtmeLHF7AuTMBoc2kkyJj0Jg0IhjqfORIQiifW0YwaWh/eEch9Kzxi+d5DGMaL')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6/mE/7xT6rZ8h11dWwHMB8K+XhqNVnzgkLEx6mFntRC/HKPGbRaqeHWiQrIPUZk+Y8eJlA4FrkI/snDyO4Gbo8OI2kef7PaPV1tiL9GVYbwPoD+/KQsb1RwMVkNMHiRhJyerMzX2e5DHOBZ8a9/gtY5QROKq17OF9I6WAbW4Kt+oyAMvwPhvr53K3PAgkUZZO')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6/mE/7xT6rZ8h11dWwHMB8K+XhqNVnzgkLEx6mFntRC/HKPGbRaqeHWiQrIPUZk+Y8eJlA4FrkI/snDyO4Gbo8OI2kef7PaPV1tiL9GVYbwPoD+/KQsb1RwMVkNMHiRhJyerMzX2e5DHOBZ8a9/gtY5QROKq17OF9I6WAbW4Kt+oyAMvwPhvr53K3PAgkUZZO')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6Xph2n4tSYT7iRhBsLn99bykQNuFI8oWckzuWcF5nUbl2GcJ53n32YnMSNtQLfuyt0oCjSSXS7BBq9uQSPQKWANBAN6MynSQzQ3UIEsxq6ExtdE1Ua22umxmxeo8vn/xYN6CaLnl0Bji1V7HlOztzRpZSml7ZVoNtbMf8iwThdOj4XR3EMElcHowQY2zd+Tzn')
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
WalletState(sequence=8, encrypted_wallet='czo4MTkyOjE2OjE6xxIydbWzxcZ2e7OivUevFO98qzS/Fy/bag0IN5/Ecm8GDEgEY84deAli9YiVxCTbwuMM1qAaL9wuC/Rj8fU9FykmAa8YEghEfIiuOTPyaySgSvDp2JY6gdZ+N+fx7qkJfzXshz5q5TuMdztWCouh4sCoaV2c+Gl7ieijq6A0c4lccOTUur+LX1mrEC5KP9Zs')
'Success'
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

The client that changed its password can easily get a new token because it has the new password saved locally. The other client needs to update its local password first.

```
>>> c1.get_auth_token()
Got auth token:  3d5227f7873a43fecb991e5026d413142541720f33ca92898acf2c8b1cdeb20d
>>> c2.get_auth_token()
Error 401
b'{"error":"Unauthorized: No match for email and password"}\n'
>>> c2.set_local_password("eggsandwich")
Generating keys...
Done generating keys
>>> c2.get_auth_token()
Got auth token:  d062d33d5692f7466f5560560fecc8e17fb903f13f5be8e4289a48a395a4306b
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
WalletState(sequence=9, encrypted_wallet='czo4MTkyOjE2OjE6i5NbYdtHfFjeIBfN1EL2nmOGlCr6hFbbPI5Y8Eq2JNeWDDy4UXTGJRNMA0SamvxneDb09RpwrW6+ffEo931rdZx0dozHCkEjKTeV5gthzbdoA7FXbiyDpJnx8DDyw2wyV/PjDKbH3dL2ojr/EgfiFivLq3FLXzopclXlL9zSipdKL3qgzN7PfRWuqiiNoY8q')
'Success'
>>> c1.change_password("starboard")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=10, encrypted_wallet='czo4MTkyOjE2OjE6GCvOav9loezTMQiq9KD7eQ834lIcOsVum6+zakt/GAX7t527dYbQ8HFWSB2O0CGuD4R5j4P2AJC7tIhBmhNbWGeAXjDtxlDFDRE//9BsFkZLAUyxGcMaOPz/obFXNrO0lFGM456fSS1E6EX17gmkDT1T6DKPQd9oNwx8UteEBLNz8V2Cw8Aa/eBrtzlgCSMf')
'Success'
```
