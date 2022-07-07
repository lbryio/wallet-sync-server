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
Got auth token:  d7159a5a84d97cdb02c642ad5d866ebfb5f69e390c873591a5620e3614d0bdeb
>>> c2.get_auth_token()
Got auth token:  9170ffc5ec3a581623bb3b17efcce3d261cb6ab480be1b295c25921fdfd8bd3c
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6EMrhL3U4HXiCJZbTyV3fAgA5XFG3C0Qwak7t/g1QBQHjpztK98587mSN5e+MNJ3+a9ydQi9q+piyLqA79WjsnyOsUKJTlGbrsVXqEzJo/mvUdb4HtDa0MaK2arvl8RD+hqsiqP/G5PVOO0JIfl3A15QHbA5/GsY8zG3xqQK95Zg=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6EMrhL3U4HXiCJZbTyV3fAgA5XFG3C0Qwak7t/g1QBQHjpztK98587mSN5e+MNJ3+a9ydQi9q+piyLqA79WjsnyOsUKJTlGbrsVXqEzJo/mvUdb4HtDa0MaK2arvl8RD+hqsiqP/G5PVOO0JIfl3A15QHbA5/GsY8zG3xqQK95Zg=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6hb2Qt8BCiLujMT0ykatcAvuVhW7uMuVbhSONFhQkLQwhZ+qBlYyTxmLc+tGzwmRLPYWCbfWY+oDcHE90h9wEKwGjNyDdjybkBRPvt4ufyIyYV/a3UPCvVYgdvktBRUF8fBagTQR2V/FQwXEeNYAAx53YSQQfy7FTYnjT2wVlbww=')
'Success'
>>> c1.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6hb2Qt8BCiLujMT0ykatcAvuVhW7uMuVbhSONFhQkLQwhZ+qBlYyTxmLc+tGzwmRLPYWCbfWY+oDcHE90h9wEKwGjNyDdjybkBRPvt4ufyIyYV/a3UPCvVYgdvktBRUF8fBagTQR2V/FQwXEeNYAAx53YSQQfy7FTYnjT2wVlbww=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6Boe8tNSgXWgoCDHDdCaLWauz6UEF2NqgjvdszFkFEOEgRIg3TYSIM2IncYG6JzeY/jjcSVAdARZ2mhW2qu2w42O2KUR53B7272YCohRUQjTG2VGj3r8idt6RF3gdJz4kPTvj9Mb2hHgxLLEsmpGrH5sAoVtnctP4kkbw4tt9yTMenzxBf330eN0kBikHMRDS')
'Success'
>>> c2.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6Boe8tNSgXWgoCDHDdCaLWauz6UEF2NqgjvdszFkFEOEgRIg3TYSIM2IncYG6JzeY/jjcSVAdARZ2mhW2qu2w42O2KUR53B7272YCohRUQjTG2VGj3r8idt6RF3gdJz4kPTvj9Mb2hHgxLLEsmpGrH5sAoVtnctP4kkbw4tt9yTMenzxBf330eN0kBikHMRDS')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE62nX6KIGR6GewHaJNGyA4hgu8Ce4mX6RTTjEHZE1NJ+ABlxz88639N/56ybBHIN8Ztcb33kLcsz+YWxn5esLVkjoEl49It6VK5mIFkUtL9QVGvMaFExUD3+l7v6USq3U92Aulu/l20WB2ZV0IqXZ7KX+GN54Yez/Vv9diQwyUujZa5n5+yoU7sY45rQ0xwmTS')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE62nX6KIGR6GewHaJNGyA4hgu8Ce4mX6RTTjEHZE1NJ+ABlxz88639N/56ybBHIN8Ztcb33kLcsz+YWxn5esLVkjoEl49It6VK5mIFkUtL9QVGvMaFExUD3+l7v6USq3U92Aulu/l20WB2ZV0IqXZ7KX+GN54Yez/Vv9diQwyUujZa5n5+yoU7sY45rQ0xwmTS')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6yRz92fRLp8UCOZ0jfNwkY2ZnCS5DSdzt06++co48MSWvKLhflrjpqBbwup4QWHB9O+1VAKoi2KPB0fbIHrnTeXLzHXkN6lPWUyOsVg61JP37FsPQBdOf7smdeImzh6bj5AT7N6qltsdYa6OdGsA2+K7syS/NJsnAE2pXLuNZWGJkDgkThH6zMiBayX2HpDeh')
'Success'
>>> c1.get_remote_wallet()
Got (and maybe merged in) latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6yRz92fRLp8UCOZ0jfNwkY2ZnCS5DSdzt06++co48MSWvKLhflrjpqBbwup4QWHB9O+1VAKoi2KPB0fbIHrnTeXLzHXkN6lPWUyOsVg61JP37FsPQBdOf7smdeImzh6bj5AT7N6qltsdYa6OdGsA2+K7syS/NJsnAE2pXLuNZWGJkDgkThH6zMiBayX2HpDeh')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE67VjoKcDba0+yJBoEasS8RKGHH8c7JbShgv+lf3CVnHXPFsA45Y3zmvyLEIsvpUxmg/jE5rw/jsh1ZCNt/yKOjRhyR8VFwR69hPl3n5j+2ya1tu4G++7REfriAkRw4kHP1im5NJ0WXPMIvdM2bV+nTFqLMdqxySyF1ljsXEdhtu9cw8A4Qs1DYOPPKfewtHNF')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE67VjoKcDba0+yJBoEasS8RKGHH8c7JbShgv+lf3CVnHXPFsA45Y3zmvyLEIsvpUxmg/jE5rw/jsh1ZCNt/yKOjRhyR8VFwR69hPl3n5j+2ya1tu4G++7REfriAkRw4kHP1im5NJ0WXPMIvdM2bV+nTFqLMdqxySyF1ljsXEdhtu9cw8A4Qs1DYOPPKfewtHNF')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6+PrieMsaswjsA5TXASYa2MwLHJEYHCAypDagR95NmAVI7/SefVs8aF1s7mA/CMTLiV3N1qwyzLMXpOxSbEiBLvjgOL00ajrHLw/ZPmOOToFIul4/9Jw5mTnqisdRWBaAF2yzXsflY2zQFllmSBJPRAiWAZ0xaErW+SJhKZzHK/aBg2PC9v1GFR7lZXpqx4CQ')
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
WalletState(sequence=8, encrypted_wallet='czo4MTkyOjE2OjE6/S99ffv8LCcu1Xk9jLjROv4tQ/nUJnxkazOfVg+eTCBOB0WiGvPKTPzo4QkmpDsNa4N3ZHHIFfwz+xG4Q+xMuWlBU38Ok6Igqtc/du6/IzQxRkUumm49s8xaFeFqQB+mqawq89RB9UDMjYlzvSDPD7ZAgxpCXkT5oIuIqkGqnc9XlIStDAlisIfNs67Orrja')
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
Got auth token:  fc40aea0ba6193f0c1903c0c95ed27010a50cea5176c75813c90ff9eb56996f8
>>> c2.get_auth_token()
Error 401
b'{"error":"Unauthorized: No match for email and password"}\n'
>>> c2.set_local_password("eggsandwich")
Generating keys...
Done generating keys
>>> c2.get_auth_token()
Got auth token:  9230030eac6107b889e82d2abb0d4be189680d3882a44e055ac7a1c9db30e00b
```
