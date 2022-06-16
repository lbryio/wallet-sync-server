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
Got auth token:  2e1c00c0f2f205defc177bd21e64dd01c669e234cf23bbc19f73e720ac1ef12d
>>> c2.get_auth_token()
Got auth token:  07ab32cfac3961d30570537d4082abdf08123de6e5a28670dbf680be45e442d5
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
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6CjZHlCv4ZyHiPKA7PoIOOkQ6Fh9fYUYPe9xwiZRYdLKHDgtEQCIcwkNldP1TN8TwTht4Qj5QEnApwQkd2Y20nVjdCUTKLzu4gmdP8QBz2EEGR+XmZgosX937E8bmmqgC55ttgt8fh0o62cTonF4h1LLI7DoWw1SvEcqIIAEn/dc=')
'Success'
```

Now, call `init_wallet_state` with the other client. This time, `init_wallet_state` will GET the wallet from the server. In general, `init_wallet_state` is used to set up a new client; first it checks the server, then failing that, it initializes it locally. (In a real client, it would save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6CjZHlCv4ZyHiPKA7PoIOOkQ6Fh9fYUYPe9xwiZRYdLKHDgtEQCIcwkNldP1TN8TwTht4Qj5QEnApwQkd2Y20nVjdCUTKLzu4gmdP8QBz2EEGR+XmZgosX937E8bmmqgC55ttgt8fh0o62cTonF4h1LLI7DoWw1SvEcqIIAEn/dc=')
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6LsWo7O3EQVw+buxGPuqJBBEn3oBM3/sAII2NjpbKi7tEvWxbWmKb+nNyr3fuvQ6YZZda0i0Rb7Veuq7ym+hYAn2pTt/8WrYR8K1HFnxs3y1m91HQIsXrl6NwxU5t+mZ6uInQUfEGEV6JLHfbt1NJ2pYlYvYTelusKZXq/kja8i4=')
'Success'
>>> c1.get_remote_wallet()
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6LsWo7O3EQVw+buxGPuqJBBEn3oBM3/sAII2NjpbKi7tEvWxbWmKb+nNyr3fuvQ6YZZda0i0Rb7Veuq7ym+hYAn2pTt/8WrYR8K1HFnxs3y1m91HQIsXrl6NwxU5t+mZ6uInQUfEGEV6JLHfbt1NJ2pYlYvYTelusKZXq/kja8i4=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6l5SVvs2yNDoC5j1316n0xQ5H6K1UEso/JpdpShfLW2bCY3lg9vOcwayO1v085RyItxEwtrtSnD3fnan3kr86GmSI8U6x5DxASHVdgceLBrclVkuCpFXllz6YNtWo5thjbf63PWSg4k6LHI8w50BT2tu9FUufCi67n7sTWnGb/0AjAFYU1sUTJ9aoeiZYrrur')
'Success'
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6l5SVvs2yNDoC5j1316n0xQ5H6K1UEso/JpdpShfLW2bCY3lg9vOcwayO1v085RyItxEwtrtSnD3fnan3kr86GmSI8U6x5DxASHVdgceLBrclVkuCpFXllz6YNtWo5thjbf63PWSg4k6LHI8w50BT2tu9FUufCi67n7sTWnGb/0AjAFYU1sUTJ9aoeiZYrrur')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE66nridrsrXcL/DlcudUs7RaAIZ3jRYQJhaacRH3vPNx0TZqkJbDcjMiHbiHY6U2AVhoAsLPIf/zcU+uDTw4IRcOL9Gozupc8tCrIcgm/kwXjnQI9RNzIfDsFxalBKj0u7Xf0c+5f/ntr4Hs9Q/Y7qthseNbUBZKU12KxNlmDcE7knLOui6NQdsUvFpuI/Rzgr')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE66nridrsrXcL/DlcudUs7RaAIZ3jRYQJhaacRH3vPNx0TZqkJbDcjMiHbiHY6U2AVhoAsLPIf/zcU+uDTw4IRcOL9Gozupc8tCrIcgm/kwXjnQI9RNzIfDsFxalBKj0u7Xf0c+5f/ntr4Hs9Q/Y7qthseNbUBZKU12KxNlmDcE7knLOui6NQdsUvFpuI/Rzgr')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE68NAahtUE4gg2M6Fam/E3brb4sv1TzcXJLvGRh4CY4416haOF1lxmKSdrvIPpOBvpNPS0B5qCbmpaKQ8Pm/WRCLj1yYUDVKgSZx0ru7AJBHiBLtpKA99Ia7XlWl129p6WtjJkbOoW8Ya+PEii72g4nrtM+j40Xe9UbVI463tlKYaRvmKr/BcoFGMJSB10Whh8')
'Success'
>>> c1.get_remote_wallet()
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE68NAahtUE4gg2M6Fam/E3brb4sv1TzcXJLvGRh4CY4416haOF1lxmKSdrvIPpOBvpNPS0B5qCbmpaKQ8Pm/WRCLj1yYUDVKgSZx0ru7AJBHiBLtpKA99Ia7XlWl129p6WtjJkbOoW8Ya+PEii72g4nrtM+j40Xe9UbVI463tlKYaRvmKr/BcoFGMJSB10Whh8')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6sh95Bt0OfcDY3QwUWaPgPWD1WPYCkN2yg1+XLD/5puONhNyjzVAnhINqVvPy52pxfkVgkIScLacMQFq4W19d+SC5LConu+fPchBzYj14Wvc3/IEQiQIxbmkv6N9USvYsjAzjGqK7szistRJY4MHC4/wRbWRprfIE7BFcDaisFSe18mRs8D2KlhEzjNJu+X8+')
'Success'
>>> c1.update_remote_wallet()
Wallet state out of date. Getting updated wallet state. Try posting again after this.
Got new walletState:
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6sh95Bt0OfcDY3QwUWaPgPWD1WPYCkN2yg1+XLD/5puONhNyjzVAnhINqVvPy52pxfkVgkIScLacMQFq4W19d+SC5LConu+fPchBzYj14Wvc3/IEQiQIxbmkv6N9USvYsjAzjGqK7szistRJY4MHC4/wRbWRprfIE7BFcDaisFSe18mRs8D2KlhEzjNJu+X8+')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
```

Now that the merge is complete, the client can make a second POST request containing the merged wallet.

```
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Got new walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE68J19IGGfoiRDm15Nb1sTj5yP9Mc3jpAeYarh206kLXKMKLKCIahmhLDMqBCXgwDe098uaIqB6IwKDfXbCVJHhWfqzu/5GoWPK1QZjhCu0rGxteFv4Tio0IYGg8CUYCvOhpQA319SXEf4sF9cyC32VwlL6qkJ2TzWTu9bTGUfZRGV3q9Rt9oL4OQHxuNIPEiE')
'Success'
```
