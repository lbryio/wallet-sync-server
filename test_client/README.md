# Test Client

A couple example flows so it's clear how it works.

## Initial setup and account recovery

Set up two clients with the same account (which won't exist on the server yet).

```
>>> from test_client import Client
>>> c1 = Client()
>>> c2 = Client()
>>> c1.set_account("joe2@example.com", "123abc2")
>>> c2.set_account("joe2@example.com", "123abc2")
```

Register the account on the server with one of the clients.

```
>>> c1.register()
Registered
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  a489d5cacc0a3db4811c34d203683482d90c605b03ae007fa5ae32ef17252bd9
>>> c2.get_auth_token()
Got auth token:  1fe687db8ab493ed260f499b674cfa49edefd3c03a718905c62d3f850dc50567
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. This increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

Note that after POSTing, it says it "got" a new wallet. This is because the POST endpoint also returns the latest version. The purpose of this will be explained in "Conflicts" below.

```
>>> c1.new_wallet_state()
>>> c1.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=1, encrypted_wallet='-')
```

With the other client, GET it from the server. Note that both clients have the same data now.

```
>>> c2.get_wallet()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='-')
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=2, encrypted_wallet='-')
>>> c1.get_wallet()
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='-')
```

## Wallet Changes

For demo purposes, this test client represents each change to the wallet by appending segments separated by `:` so that we can more easily follow the history. (The real app will not actually edit the wallet in the form of an append log.)

```
>>> c1.cur_encrypted_wallet()
'-'
>>> c1.change_encrypted_wallet()
>>> c1.cur_encrypted_wallet()
'-:cfF6'
```

The wallet is synced between the clients.

```
>>> c1.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=3, encrypted_wallet='-:cfF6')
>>> c2.get_wallet()
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='-:cfF6')
>>> c2.cur_encrypted_wallet()
'-:cfF6'
```

## Merging Changes

Both clients create changes. They now have diverging wallets.

```
>>> c1.change_encrypted_wallet()
>>> c2.change_encrypted_wallet()
>>> c1.cur_encrypted_wallet()
'-:cfF6:565b'
>>> c2.cur_encrypted_wallet()
'-:cfF6:6De1'
```

One client POSTs its change first.

```
>>> c1.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=4, encrypted_wallet='-:cfF6:565b')
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally.

The _merge base_ that a given client uses is the last version that it successfully got from or POSTed to the server. You can see the merge base here: `"-:cfF6"`, the first part of the wallet which both clients had in common before the merge.

```
>>> c2.get_wallet()
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='-:cfF6:565b')
>>> c2.cur_encrypted_wallet()
'-:cfF6:565b:6De1'
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=5, encrypted_wallet='-:cfF6:565b:6De1')
>>> c1.get_wallet()
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='-:cfF6:565b:6De1')
>>> c1.cur_encrypted_wallet()
'-:cfF6:565b:6De1'
```

## Conflicts

A client cannot POST if it is not up to date. It needs to merge in any new changes on the server before POSTing its own changes. For convenience, if a conflicting POST request is made, the server responds with the latest version of the wallet state (just like a GET request). This way the client doesn't need to make a second request to perform the merge.

(If a non-conflicting POST request is made, it responds with the same wallet state that the client just POSTed, as it is now the server's current wallet state)

```
>>> c2.change_encrypted_wallet()
>>> c2.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=6, encrypted_wallet='-:cfF6:565b:6De1:053a')
>>> c1.change_encrypted_wallet()
>>> c1.post_wallet()
Wallet state out of date. Getting updated wallet state. Try posting again after this.
Got new walletState:
WalletState(sequence=6, encrypted_wallet='-:cfF6:565b:6De1:053a')
```

Now the merge is complete, and the client can make a second POST request containing the merged wallet.

```
>>> c1.post_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=7, encrypted_wallet='-:cfF6:565b:6De1:053a:6774')
```
