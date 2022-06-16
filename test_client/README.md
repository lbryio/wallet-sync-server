# Test Client

A couple example flows so it's clear how it works. We're assuming that we're starting with a fresh DB on the server, and that we've created two wallets on the SDK: `"test_wallet_1"` and `"test_wallet_2"`.

## Initial setup and account recovery

Set up a client for each wallet, but with the same sync account (which won't exist on the server yet). This will simulate clients on two different computers.

```
>>> from test_client import Client
>>> c1 = Client("joe2@example.com", "123abc2", 'test_wallet_1')
>>> c2 = Client("joe2@example.com", "123abc2", 'test_wallet_2')
```

Register the account on the server with one of the clients.

```
>>> c1.register()
Registered
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  3d98076fda58400f3dbd5ea6511184507d5f8637bd5549e5cb0cc9cdbb7102e5
>>> c2.get_auth_token()
Got auth token:  1385a51bf3ba86a3e1f412908c3b2165cc399e06692a2dc602f5e17fe2c7993c
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

_Note that after POSTing, it says it "got" a new wallet. This is because the POST endpoint also returns the latest version. The purpose of this will be explained in "Conflicts" below._

```
>>> c1.init_wallet_state()
Wallet not found
No wallet found on the server for this account. Starting a new one.
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6ew8QGI/89sz70Oud6NljymaLSUCyNSBYwpTCBZu9gMbwXYuDKqB4YnZeYRJdHhXz+9NQ9qSkRUIPHQ4m6f38R38KvXCE5raRnnozrnmDOt/eGFUl9XYMrFcYqgqYSCxb1kTcWS1cWkkOO6TtrjeBKuc+qriKZr9ggk1pnLmnKQc=')
'Success'
```

Now, call `init_wallet_state` with the other client. This time, `init_wallet_state` will GET the wallet from the server. In general, `init_wallet_state` is used to set up a new client; first it checks the server, then failing that, it initializes it locally. (In a real client, it would save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6ew8QGI/89sz70Oud6NljymaLSUCyNSBYwpTCBZu9gMbwXYuDKqB4YnZeYRJdHhXz+9NQ9qSkRUIPHQ4m6f38R38KvXCE5raRnnozrnmDOt/eGFUl9XYMrFcYqgqYSCxb1kTcWS1cWkkOO6TtrjeBKuc+qriKZr9ggk1pnLmnKQc=')
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6DAT6j6JSp0by78XpOOMtGroxFUX5vh6X+oXhIVlHVhvmVgp+09vWt7IP/IGofP4Ua7Dggr9iyxF4A3F9tSNgxKrev08eMP+8W2LAwk3jTAtZPoh5vtz/20tJFWOw+Y+s00NRNXcDeT8GjZvgTfawy+k7WKQMt6ryW6c8liORDfA=')
'Success'
>>> c1.get_remote_wallet()
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6DAT6j6JSp0by78XpOOMtGroxFUX5vh6X+oXhIVlHVhvmVgp+09vWt7IP/IGofP4Ua7Dggr9iyxF4A3F9tSNgxKrev08eMP+8W2LAwk3jTAtZPoh5vtz/20tJFWOw+Y+s00NRNXcDeT8GjZvgTfawy+k7WKQMt6ryW6c8liORDfA=')
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
Got new walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6SQ/9PBDeOs8sOG+QDnOBmgbOKJDUx3TJD1p9r/bIuD2R5lamKmn1UKz/fQynLJexPJj3QCJP5u5OKTZDMBhY6HF5qBV2ndnWmPLjB40KlGj7jjZJaETEMktyJjjKdLbsV8nKLpnB2KpyYZejJVppBS+DRswAFByTE6c5E+8FJ3TTPXhzTvE9L3RqvetQEUxn')
'Success'
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6SQ/9PBDeOs8sOG+QDnOBmgbOKJDUx3TJD1p9r/bIuD2R5lamKmn1UKz/fQynLJexPJj3QCJP5u5OKTZDMBhY6HF5qBV2ndnWmPLjB40KlGj7jjZJaETEMktyJjjKdLbsV8nKLpnB2KpyYZejJVppBS+DRswAFByTE6c5E+8FJ3TTPXhzTvE9L3RqvetQEUxn')
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
Got new walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE62uWympFofMnLmZSGGPTC5qctGKlWkan/DmOFLVZHktzqY9OndxhY3VCr5QBMXOGyn/Y321zNtL6YNfA+gs3Ov6qhzcneERHJM3ByySjMPwEds4NVDctKW4HAXggZIA1xhga1XlNggrBXlu09Sqro9zEbJdrBwJQI6BeuZHpH2eaJBDI73ljTWgtqoIeLg1WA')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE62uWympFofMnLmZSGGPTC5qctGKlWkan/DmOFLVZHktzqY9OndxhY3VCr5QBMXOGyn/Y321zNtL6YNfA+gs3Ov6qhzcneERHJM3ByySjMPwEds4NVDctKW4HAXggZIA1xhga1XlNggrBXlu09Sqro9zEbJdrBwJQI6BeuZHpH2eaJBDI73ljTWgtqoIeLg1WA')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6ngb8TU1FyKgmzyHLQ8c30yOg/kVFNSDbtquXHKs16vEIQta3zrJLnGiY0WoiXx8Ul4uvYLK1riNaoo+OfZYtJjtpYLWf1oGdn0PDq0ZCHhK6GcX2Zbz/YQEdPcOvDeENjxZ4Pq2qoZYSDcPvwOgbvO2FSOK27OhCWHCA/9LbzAu6Suq6RS3i2p2TpmUHtz2H')
'Success'
>>> c1.get_remote_wallet()
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE6ngb8TU1FyKgmzyHLQ8c30yOg/kVFNSDbtquXHKs16vEIQta3zrJLnGiY0WoiXx8Ul4uvYLK1riNaoo+OfZYtJjtpYLWf1oGdn0PDq0ZCHhK6GcX2Zbz/YQEdPcOvDeENjxZ4Pq2qoZYSDcPvwOgbvO2FSOK27OhCWHCA/9LbzAu6Suq6RS3i2p2TpmUHtz2H')
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

We try to POST both of them to the server, but the second one fails because of the conflict. Instead, merges the two locally:

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6MnLcl2+VTv8B9MIKJjpwptjF8Ws6NfhFkFBnsTDy8arv7akMSV/jojkvz2bJzOjX+iAKiY0+FKgD2akONsUnQqF95pnbr+TPnpbFxS4TLFUWxbpJMm7+r3FZiOauMZ6ewBfBq3vzI2UA2o3RrSxzucKZ6ZcgZqJsKCnk+rCj/ADmrUJb01kwB6WDATcMlG5A')
'Success'
>>> c1.update_remote_wallet()
Wallet state out of date. Getting updated wallet state. Try posting again after this.
Got new walletState:
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6MnLcl2+VTv8B9MIKJjpwptjF8Ws6NfhFkFBnsTDy8arv7akMSV/jojkvz2bJzOjX+iAKiY0+FKgD2akONsUnQqF95pnbr+TPnpbFxS4TLFUWxbpJMm7+r3FZiOauMZ6ewBfBq3vzI2UA2o3RrSxzucKZ6ZcgZqJsKCnk+rCj/ADmrUJb01kwB6WDATcMlG5A')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
```

Now that the merge is complete, the client can make a second POST request containing the merged wallet.

```
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6uexO9yl0JVsKFo6WeJGOsJ/sm1RJPc+NwLxniaE744lVEihK2HyNxDVbcAFEMxn/vKXgFKtzLV/D7eAzeGrSIyQR5v3YeZrTWPcRzK79rJgHzjcZpjKpytcDMZp2lB5cRHkNg7u8qAa2DnbebMXU0CKblTL++IIteU+CzyuTdW1Uoj4cEOsy6G8HwrZc5drf')
'Success'
```
