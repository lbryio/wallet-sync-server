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
'09f355b525ad440c9fdd0e95e5ac2c49708f84112772ea322c28311c53d55410'
```

Set up the other client. See that it got the same salt seed from the server in the process, which it needs to make sure we have the correct encryption key and login password.

```
>>> c2.update_derived_secrets()
Generating keys...
Done generating keys
>>> c2.salt_seed
'09f355b525ad440c9fdd0e95e5ac2c49708f84112772ea322c28311c53d55410'
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  be540cea8c18d9de40255ab8da4cbc6d561512731f38e36e808903234453ba4d
>>> c2.get_auth_token()
Got auth token:  4cde9acc071e25a4fdebcd5a019d84d4f38c7093e4d15ddda5282d0ce136278d
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6FkHI5eVQvbw7vEouMyn766uHhWJBskmh7ZWNttSHh6Ro0ETPZXXuIuPMM+9/1e6Maq+CFBToS2mI/7Z6fuGdVRAGLXVNZKXY9g48J9R7TikFas/XE1Vs/bKLKGKAgvlhJCYvzg582z7JrMzaAQ26VJwFL0gbx+ey+wo7ukb5Yz4=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6FkHI5eVQvbw7vEouMyn766uHhWJBskmh7ZWNttSHh6Ro0ETPZXXuIuPMM+9/1e6Maq+CFBToS2mI/7Z6fuGdVRAGLXVNZKXY9g48J9R7TikFas/XE1Vs/bKLKGKAgvlhJCYvzg582z7JrMzaAQ26VJwFL0gbx+ey+wo7ukb5Yz4=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6c1dN5L/ywTpgtRIs35zwdPubo+rkXiuk6yoQdw63c/pW+X2NWuFhLo0sfITnRvjiyK20fIwGmBK2nnp2BllEV41HUHETitLL+HS5KK8+/ReNaqE/bZQs1pDn7K+RBl14heAPQQXLeFNFp9fNZxPGs1N+uLl0sRnhij+8kjS5p0Q=')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6c1dN5L/ywTpgtRIs35zwdPubo+rkXiuk6yoQdw63c/pW+X2NWuFhLo0sfITnRvjiyK20fIwGmBK2nnp2BllEV41HUHETitLL+HS5KK8+/ReNaqE/bZQs1pDn7K+RBl14heAPQQXLeFNFp9fNZxPGs1N+uLl0sRnhij+8kjS5p0Q=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6utvO9D4f/H8EAcRi1RrsxwnF556Mb/nE77yD98i2Ml75Y0FynfjJYho05n2l4RZDirV5Bm4KrXLH3D+qZh3hqW+ejTvMTUYJ2pLv9jNx7ZuNtPCF6tn+LaRur9sfy+jyuxmeHR9Wd7rqimg5pIBed2DLUTA5R5QaG7TEiMYTrUN+RrkYFdANEvh8LOgRDpce')
'Success'
>>> c2.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6utvO9D4f/H8EAcRi1RrsxwnF556Mb/nE77yD98i2Ml75Y0FynfjJYho05n2l4RZDirV5Bm4KrXLH3D+qZh3hqW+ejTvMTUYJ2pLv9jNx7ZuNtPCF6tn+LaRur9sfy+jyuxmeHR9Wd7rqimg5pIBed2DLUTA5R5QaG7TEiMYTrUN+RrkYFdANEvh8LOgRDpce')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE65B/Ia0+1qF1fKqsAfTcf4tx6tJK95yCGcuCVRNk83URaksesq+qI/pqtPX2NOwX6y3KJhO75McHI1Wg5AJx3NkA/jV6VeFVHnAJu0UpkzqI1RtTMEn3w63VXQzLmmj+8PeEEdbY5f5/59fuUhTKfrPLTbr8dSvKIsKQC36d17oGhI/Ox8eQqS5QdjeoARENT')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Merging local changes with remote changes to create latest walletState.
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE65B/Ia0+1qF1fKqsAfTcf4tx6tJK95yCGcuCVRNk83URaksesq+qI/pqtPX2NOwX6y3KJhO75McHI1Wg5AJx3NkA/jV6VeFVHnAJu0UpkzqI1RtTMEn3w63VXQzLmmj+8PeEEdbY5f5/59fuUhTKfrPLTbr8dSvKIsKQC36d17oGhI/Ox8eQqS5QdjeoARENT')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6p7gs7r4Q6ky7/93vf/1e3HPXTKSAPP6c+0oxYl7010WKqkT0m9uVzU/YZW2QdCrPO4M99XiuxjpnbKwl42BnrvIkpfO1JHaDQHYTVZjp5W7UZHeevR8zQiyXWpEIDdTt1dwwKvpP7RHN1RqJzbyTCfJ6YqA7/GDiAojfDyz/5CvsIxcMCI+vTgYDwcy4rP36')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6p7gs7r4Q6ky7/93vf/1e3HPXTKSAPP6c+0oxYl7010WKqkT0m9uVzU/YZW2QdCrPO4M99XiuxjpnbKwl42BnrvIkpfO1JHaDQHYTVZjp5W7UZHeevR8zQiyXWpEIDdTt1dwwKvpP7RHN1RqJzbyTCfJ6YqA7/GDiAojfDyz/5CvsIxcMCI+vTgYDwcy4rP36')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6JFuPERZ3SB2CiSzPISB5/KteOvn1348Xk1o+sgFQMO8l1woKG7pRFUH89L4cm9q/521lQsXgujwEW30X8XQnoTEC0VzaM7IWI7yug7uRQ1XhpXd5W97L3tTlAnZ+V+/lg76Auk9COkJYrbQ8raon3qNl0np9A8pQLvbxVTYAvoA8AMzMEUBFeuZoADZX6bJI')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6JFuPERZ3SB2CiSzPISB5/KteOvn1348Xk1o+sgFQMO8l1woKG7pRFUH89L4cm9q/521lQsXgujwEW30X8XQnoTEC0VzaM7IWI7yug7uRQ1XhpXd5W97L3tTlAnZ+V+/lg76Auk9COkJYrbQ8raon3qNl0np9A8pQLvbxVTYAvoA8AMzMEUBFeuZoADZX6bJI')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6ZnqsMEkm/XFvZzRQ1RL6WKw4CeGJeLFp49gnE2SlZYaMjtnels+Cl6EDE1vlGF42bju0lUFLEEVwa1JSweu11Q0IIZDKbjqDLChMCLLItf7aEAxJsZ4/3nu/N2BVCK8Hxz81p0UJiExUImI84Q4FWRqMPgnal8A/OeW7Qor9IyuqkuP054Lt3GnYeJYd3mfN')
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
WalletState(sequence=8, encrypted_wallet='czo4MTkyOjE2OjE6bk6pmqtNnjBYgHvwo2wAVYDwD/Un0hTdbjSJBoRAJBAo6ZYk9VCjlEpPtrOo7UZiP7KgZpDI3KjI00dqG019krTalmuVKp3gXgojlK99Z9ZGSXQRK5baehq4cRmkuAaqDGysajRsVaWbPtNUgLM6Dw4FfoveUCmuATDrO+kv9Mg1tvs3a+IpMwdVXdxU2FEd')
'Success'
```

We generate a new salt seed when we change the password

```
>>> c1.salt_seed
'e081028900b64dffeb43b44c6e4beacce59510eeb78eea5df420297c77057586'
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
Got auth token:  ddbcbfb622b0b3653909f28f0b2df81e2f4c61b5f5316cf85a67e09e5e96b90c
>>> c2.get_auth_token()
Error 401
b'{"error":"Unauthorized: No match for email and/or password"}\n'
Failed to get the auth token. Do you need to verify your email address? Or update this client's password (set_local_password())?
Or, in the off-chance the user changed their password back and forth, try updating secrets (update_derived_secrets()) to get the latest salt seed.
>>> c2.set_local_password("eggsandwich")
Generating keys...
Done generating keys
>>> c2.salt_seed
'e081028900b64dffeb43b44c6e4beacce59510eeb78eea5df420297c77057586'
>>> c2.get_auth_token()
Got auth token:  f85b645bcd5d8bddd266cd95f6127dafec5f958ceab193bde0caf4767daafb71
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
WalletState(sequence=9, encrypted_wallet='czo4MTkyOjE2OjE6BggwBmly+7071Vb6oEQSjviE3a3o/o0W34ikaRwggJv37p1D3SAFEBBtPNpc0FSTIltg5f0sxd55tSwcYGUWt3uPsZSqPOuMiFYPtUtMpzg2dAi9g42wSM+m9EdlYGpKxypijqzGZD/Ck7t7PE5eDKLAUxF/voJoOg3uJMGhyzq0+j68KuSofgvCGBTwJOro')
'Success'
>>> c1.change_password("starboard")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=10, encrypted_wallet='czo4MTkyOjE2OjE66qYzTZ+M37RQUfSRTKfVv9RvbqewMaQd6ZBsjZD+dvvjtXqcB6ZZ4q5K7Gd09VMkHRCP8tCYZ/48b/Yvt5sDiTcWSh/Ou08aaiUChPERKWNLpqOCTuF2B/DbsQPgCjwUiXNan+tW6rMbFM6QHxJ/FIzkJUsmab3nwYPon2xkRY8pnwj02R7J62Y7lMd7RiYV')
'Success'
>>> c1.get_auth_token()
Got auth token:  deb6a1a77ca601d01a75c6f150a767b6038e68cfb6cfd806333e5f5b6f03870f
>>> c2.set_local_password("starboard")
Generating keys...
Done generating keys
>>> c2.get_auth_token()
Got auth token:  d7fb866d4e77426a9c9c32f99a71bd73de53370ec4cc836e734461b4910ab66f
```

# Websockets

A client can make a websocket connection to the server and receive notifications whenever another client updates the wallet on the server. The message will contain the sequence number so that the client can know whether they happen to be up to date. (The client that made the update will of course be up to date).

This test client will have a thread listening to the websocket which just prints info about new messages as they come in. A real client would likely choose to get the latest wallet from the server as soon as a messag ecomes through, assuming the sequence is newer than what the client has.

```
>>> c1.start_websocket()
>>> c2.start_websocket()
```
c1 connected for now
c2 connected for now

Now make an update and see:

```
>>> c1.update_remote_wallet()
c1 got notified of a wallet update, sequence=11. If your client is behind this sequence, you should get the latest from the server.
Successfully updated wallet state on server
c2 got notified of a wallet update, sequence=11. If your client is behind this sequence, you should get the latest from the server.
Synced walletState:
WalletState(sequence=11, encrypted_wallet='czo4MTkyOjE2OjE6unyTAofXJh02g0GNXHYjNseaZVxmsRy8zVMBrgzLRKpw1gjmhEB+tsqFl+j1Boy0aqoKGoXFM/q0LZrrDx52elZZPTciTQ5w52IVbeWKe7eYPp2R6ZDsjxNTDfCuVz/CaE/k2uC1f/AW/d6BahTjUYH37EQ3RXEYpucpqnSFKye1PkhloVEu8w5AcnwWm8s/')
'Success'
```

Update again and we'll see the new sequence number:

```
>>> c1.update_remote_wallet()
c1 got notified of a wallet update, sequence=12. If your client is behind this sequence, you should get the latest from the server.
c2 got notified of a wallet update, sequence=12. If your client is behind this sequence, you should get the latest from the server.
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=12, encrypted_wallet='czo4MTkyOjE2OjE66PTstwDb8W2Zjz/UkhzypUN8lKAZ3zSW43I3Got4+XjHQnSd+ssJ2s+6OTlAcQxoC3ngtvpV445Hcs/IEDoX00BeYtR0y5Wh0PD2507UkYhaI58R8Ie4Bw/9z/9+DBTsMmfZ1qogV9n8bbMZNjpRYGlQMFMjWbIr58uflMoD81sKI6NUiz6dBnic2QAj6Ueu')
'Success'
```

When we change a password, just as all auth tokens are invalidated, all sockets are also disconnected.

```
>>> c1.change_password("ihatesockets")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=13, encrypted_wallet='czo4MTkyOjE2OjE62yuQuJXS1yKOu3Bk/IKDwyMw0l5lZyVuYmmMzlIV+fzdGUy3pcY0pJYaz9mH664tknjqTTCIuKO1lCd+OXCByHNl6+FlE2pTNTTm0yDJoBbGzw0mKlABi1ERZAuBy3Q0GqAneKtlkikcbjgQV2KB+pHdN9bD/b0Qey0/Zf6TMCXpVEGeAPrtJiBUpgN2Nmeb')
'Success'
```
c1 disconnected for now: code = 1005 (no status code [internal]), no reason
c2 disconnected for now: code = 1005 (no status code [internal]), no reason
