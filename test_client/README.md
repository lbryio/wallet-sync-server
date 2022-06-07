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

Each device will have a device_id which will be used in the wallet state metadata to mark which device created a given version. This is used in the `lastSynced` field (see below).

```
>>> c1.device_id
'974690df-85a6-481d-9015-6293226db8c9'
>>> c2.device_id
'545643c9-ee47-443d-b260-cb9178b8646c'
```

Register the account on the server with one of the clients.

```
>>> c1.register()
Registered
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_full_auth_token()
Got auth token:  941e5159a2caff15f0bdc1c0e6da92691d3073543dbfae810cfe57d51c35f0e0
>>> c2.get_full_auth_token()
Got auth token:  b323a18e51263ac052777ca68de716c1f3b4983bf4c918477e355f637c8ea2d4
```

## Syncing

Create a new wallet state (wallet + metadata) and post it to the server. Note that after posting, it says it "got" a new wallet state. This is because the post endpoint also returns the latest version. The purpose of this will be explained in "Conflicts" below.

The fields in the walletstate are:

* `encryptedWallet` - the actual encrypted wallet data
* `lastSynced` - a mapping between deviceId and the latest sequence number that it _created_. This is bookkeeping to prevent certain syncing errors.
* `deviceId` - the device that made _this_ wallet state version (NOTE this admittedly seems redundant with `lastSynced` and may be removed)

```
>>> c1.new_wallet_state()
>>> c1.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': '',
 'lastSynced': {'974690df-85a6-481d-9015-6293226db8c9': 1}}
```

With the other client, get it from the server. Note that both clients have the same data now.

```
>>> c2.get_wallet_state()
Got latest walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': '',
 'lastSynced': {'974690df-85a6-481d-9015-6293226db8c9': 1}}
```

## Updating

Push a new version, get it with the other client. Even though we haven't edited the encrypted wallet yet, each version of a wallet _state_ has an incremented sequence number, and the deviceId that created it.

```
>>> c2.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '545643c9-ee47-443d-b260-cb9178b8646c',
 'encryptedWallet': '',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 2,
                '974690df-85a6-481d-9015-6293226db8c9': 1}}
>>> c1.get_wallet_state()
Got latest walletState:
{'deviceId': '545643c9-ee47-443d-b260-cb9178b8646c',
 'encryptedWallet': '',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 2,
                '974690df-85a6-481d-9015-6293226db8c9': 1}}
```

## Wallet Changes

For demo purposes, this test client represents each change to the wallet by appending segments separated by `:` so that we can more easily follow the history. (The real app will not actually edit the wallet in the form of an append log.)

```
>>> c1.cur_encrypted_wallet()
''
>>> c1.change_encrypted_wallet()
>>> c1.cur_encrypted_wallet()
':2fbE'
```

The wallet is synced between the clients.

```
>>> c1.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': ':2fbE',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 2,
                '974690df-85a6-481d-9015-6293226db8c9': 3}}
>>> c2.get_wallet_state()
Got latest walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': ':2fbE',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 2,
                '974690df-85a6-481d-9015-6293226db8c9': 3}}
>>> c2.cur_encrypted_wallet()
':2fbE'
```

## Merging Changes

Both clients create changes. They now have diverging wallets.

```
>>> c1.change_encrypted_wallet()
>>> c2.change_encrypted_wallet()
>>> c1.cur_encrypted_wallet()
':2fbE:BD62'
>>> c2.cur_encrypted_wallet()
':2fbE:e7ac'
```

One client posts its change first. The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally.

The _merge base_ that a given client uses is the last version that it successfully got from or posted to the server. You can see the merge base here: the first part of the wallet which does not change from this merge.

```
>>> c1.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': ':2fbE:BD62',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 2,
                '974690df-85a6-481d-9015-6293226db8c9': 4}}
>>> c2.get_wallet_state()
Got latest walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': ':2fbE:BD62',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 2,
                '974690df-85a6-481d-9015-6293226db8c9': 4}}
>>> c2.cur_encrypted_wallet()
':2fbE:BD62:e7ac'
```

Finally, the client with the merged wallet pushes it to the server, and the other client gets the update.

```
>>> c2.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '545643c9-ee47-443d-b260-cb9178b8646c',
 'encryptedWallet': ':2fbE:BD62:e7ac',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 5,
                '974690df-85a6-481d-9015-6293226db8c9': 4}}
>>> c1.get_wallet_state()
Got latest walletState:
{'deviceId': '545643c9-ee47-443d-b260-cb9178b8646c',
 'encryptedWallet': ':2fbE:BD62:e7ac',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 5,
                '974690df-85a6-481d-9015-6293226db8c9': 4}}
>>> c1.cur_encrypted_wallet()
':2fbE:BD62:e7ac'
```

## Conflicts

A client cannot post if it is not up to date. It needs to merge in any new changes on the server before posting its own changes. For convenience, if a conflicting post request is made, the server responds with the latest version of the wallet state (just like a GET request). This way the client doesn't need to make a second request to perform the merge.

(If a non-conflicting post request is made, it responds with the same wallet state that the client just posted, as it is now the server's current wallet state)

```
>>> c2.change_encrypted_wallet()
>>> c2.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '545643c9-ee47-443d-b260-cb9178b8646c',
 'encryptedWallet': ':2fbE:BD62:e7ac:4EEf',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 6,
                '974690df-85a6-481d-9015-6293226db8c9': 4}}
>>> c1.change_encrypted_wallet()
>>> c1.post_wallet_state()
Wallet state out of date. Getting updated wallet state. Try again.
Got new walletState:
{'deviceId': '545643c9-ee47-443d-b260-cb9178b8646c',
 'encryptedWallet': ':2fbE:BD62:e7ac:4EEf',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 6,
                '974690df-85a6-481d-9015-6293226db8c9': 4}}
```

Now the merge is complete, and the client can make a second post request containing the merged wallet.

```
>>> c1.post_wallet_state()
Successfully updated wallet state on server
Got new walletState:
{'deviceId': '974690df-85a6-481d-9015-6293226db8c9',
 'encryptedWallet': ':2fbE:BD62:e7ac:4EEf:DC86',
 'lastSynced': {'545643c9-ee47-443d-b260-cb9178b8646c': 6,
                '974690df-85a6-481d-9015-6293226db8c9': 7}}
```
