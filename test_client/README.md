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
'fd7bcc62d7334fbf07aca5791783cb173e3aaef91e228f000a69e3ec8eef123e'
```

Set up the other client. See that it got the same salt seed from the server in the process, which it needs to make sure we have the correct encryption key and login password.

```
>>> c2.update_secrets()
Generating keys...
Done generating keys
>>> c2.salt_seed
'fd7bcc62d7334fbf07aca5791783cb173e3aaef91e228f000a69e3ec8eef123e'
```

Now that the account exists, grab an auth token with both clients.

```
>>> c1.get_auth_token()
Got auth token:  310077f33a9b8de99ee6c45ffbe4a06a7178683e4eb65500fc5ae26513f80928
>>> c2.get_auth_token()
Got auth token:  cdd18033dc48aeefedc278d116a6abbef7f0fb525d7ddfb2e1804a817a212c4a
```

## Syncing

Create a new wallet + metadata (we'll wrap it in a struct we'll call `WalletState` in this client) using `init_wallet_state` and POST them to the server. The metadata (as of now) in the walletstate is only `sequence`. `sequence` is an integer that increments for every POSTed wallet. This is bookkeeping to prevent certain syncing errors.

```
>>> c1.init_wallet_state()
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6KY46kZ0oRC9W8g/LVCe3V3sLdCHk0sWEBqPjzcKykl3dDJpQvRXtz8HFXlD+bgvs8M5jHw7KjJ9ODUOEq3VoSawrKyZpgc8AYIx+vC4w+q6cKC3LToxr7FlfyAoQKo9dCothik/90ySVMAPY1BBrBmQ8H46eFEoMWZ4nG2OWGew=')
'Success'
```

Now, call `init_wallet_state` with the other client. Then, we call `get_remote_wallet` to GET the wallet from the server. (In a real client, it would also save the walletstate to disk, and `init_wallet_state` would check there before checking the server).

(There are a few potential unresolved issues surrounding this related to sequence of events. Check comments on `init_wallet_state`. SDK again works around them with the timestamps.)

```
>>> c2.init_wallet_state()
>>> c2.get_remote_wallet()
Got latest walletState:
WalletState(sequence=1, encrypted_wallet='czo4MTkyOjE2OjE6KY46kZ0oRC9W8g/LVCe3V3sLdCHk0sWEBqPjzcKykl3dDJpQvRXtz8HFXlD+bgvs8M5jHw7KjJ9ODUOEq3VoSawrKyZpgc8AYIx+vC4w+q6cKC3LToxr7FlfyAoQKo9dCothik/90ySVMAPY1BBrBmQ8H46eFEoMWZ4nG2OWGew=')
'Success'
```

## Updating

Push a new version, GET it with the other client. Even though we haven't edited the encrypted wallet yet, we can still increment the sequence number.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6pNxAvxzGvZZvlSuYhSdGFmHGvWImYJlEewOnp6iUBbqTo899MrCfgvvlOzBOuKwk4vpJYcQgEkb3u2+j7bnJ18OCFEUeDzq88JuoKNw6ppdAbpw8D7MIDZP4Tf+5O8LmjxKtbiMy/ztW0nUxi4Ls8uuJ436CdF0UwaevHOAvlOE=')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=2, encrypted_wallet='czo4MTkyOjE2OjE6pNxAvxzGvZZvlSuYhSdGFmHGvWImYJlEewOnp6iUBbqTo899MrCfgvvlOzBOuKwk4vpJYcQgEkb3u2+j7bnJ18OCFEUeDzq88JuoKNw6ppdAbpw8D7MIDZP4Tf+5O8LmjxKtbiMy/ztW0nUxi4Ls8uuJ436CdF0UwaevHOAvlOE=')
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
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6FvtBsBMaRvnY6+SnQiBab/FWjzC9mgu3OHFXRTcJm3MsByZWbrzkFz6y4FrPN1cC+/Rcw11oJNydyys2ZaPl6zNYP/uxV6anEmET1hTt1+E2NzUJ2K/K4BLN7AgLBLQDM8zAwVzCTeaT6MZvjTV/slYc7NEqlfCMgeId2WpBJp+BCbTLo0SjPWhHCsQvo3Hf')
'Success'
>>> c2.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=3, encrypted_wallet='czo4MTkyOjE2OjE6FvtBsBMaRvnY6+SnQiBab/FWjzC9mgu3OHFXRTcJm3MsByZWbrzkFz6y4FrPN1cC+/Rcw11oJNydyys2ZaPl6zNYP/uxV6anEmET1hTt1+E2NzUJ2K/K4BLN7AgLBLQDM8zAwVzCTeaT6MZvjTV/slYc7NEqlfCMgeId2WpBJp+BCbTLo0SjPWhHCsQvo3Hf')
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
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6rKE6TWvZxBp0No0S1CRrjW55i6w9pE6obS+9bwR76qaBBQ40lz0Ajd/2vXO1KBAQhxEHDJ6WJLPs15SgqhVspaNXmdwR1dYEHmJ8M+PW0KLv+vZoxxeGQ/5EBdrAZIfBhmI50SPF4RzmTzKTyw3VlSdqqhCutgi6FcXP+CLlsnH6qaLgjLDLISjwSMIwBd4y')
'Success'
```

The other client pulls that change, and _merges_ those changes on top of the changes it had saved locally. For now, the SDK merges the preferences based on timestamps internal to the wallet.

Eventually, the client will be responsible (or at least more responsible) for merging. At this point, the _merge base_ that a given client will use is the last version that it successfully GETed from POSTed to the server. It's the last common version between the client merging and the client that created the wallet version on the server.

```
>>> c2.get_remote_wallet()
Merging local changes with remote changes to create latest walletState.
Got latest walletState:
WalletState(sequence=4, encrypted_wallet='czo4MTkyOjE2OjE6rKE6TWvZxBp0No0S1CRrjW55i6w9pE6obS+9bwR76qaBBQ40lz0Ajd/2vXO1KBAQhxEHDJ6WJLPs15SgqhVspaNXmdwR1dYEHmJ8M+PW0KLv+vZoxxeGQ/5EBdrAZIfBhmI50SPF4RzmTzKTyw3VlSdqqhCutgi6FcXP+CLlsnH6qaLgjLDLISjwSMIwBd4y')
'Success'
>>> c2.get_preferences()
{'animal': 'horse', 'car': 'Audi'}
```

Finally, the client with the merged wallet pushes it to the server, and the other client GETs the update.

```
>>> c2.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE62Tub/EfwCMdYpZ9N9wCRwTB3Di+eA0oVpn44v/n1UgOB8jNEIEtQptCfBtBE7yfIJP8pw544SkhxAfR2Zy8/UrLIhKMUSVeCl8bJP78AoJCPpeJEQo4GOqPvluWYS2eOh1urZojn5yqB5nGRnK4hYhQ6lOwgg4jfRFtTzMKPYb263ONb3mx1SkeoCwmBeRoF')
'Success'
>>> c1.get_remote_wallet()
Nothing to merge. Taking remote walletState as latest walletState.
Got latest walletState:
WalletState(sequence=5, encrypted_wallet='czo4MTkyOjE2OjE62Tub/EfwCMdYpZ9N9wCRwTB3Di+eA0oVpn44v/n1UgOB8jNEIEtQptCfBtBE7yfIJP8pw544SkhxAfR2Zy8/UrLIhKMUSVeCl8bJP78AoJCPpeJEQo4GOqPvluWYS2eOh1urZojn5yqB5nGRnK4hYhQ6lOwgg4jfRFtTzMKPYb263ONb3mx1SkeoCwmBeRoF')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6ZNErNr5SrgjRMOBmK2pKtU2wu+jdwR8WO/thAf+VrGJ9237sKTjNX0aQILuj9dOzY836xYk2vB1Niypgf4PvlnXEAZ64pHO2FV8aR/0JcjsufkdUXUIJH2hxDhT5Ui8kS2tXPAuo0xDxfqQgqiJaVNfgyCo2fzqz5m5V3jBzivm7fN8TpuNaT94koI2GFPc3')
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
WalletState(sequence=6, encrypted_wallet='czo4MTkyOjE2OjE6ZNErNr5SrgjRMOBmK2pKtU2wu+jdwR8WO/thAf+VrGJ9237sKTjNX0aQILuj9dOzY836xYk2vB1Niypgf4PvlnXEAZ64pHO2FV8aR/0JcjsufkdUXUIJH2hxDhT5Ui8kS2tXPAuo0xDxfqQgqiJaVNfgyCo2fzqz5m5V3jBzivm7fN8TpuNaT94koI2GFPc3')
'Success'
>>> c1.get_preferences()
{'animal': 'beaver', 'car': 'Toyota'}
>>> c1.update_remote_wallet()
Successfully updated wallet state on server
Synced walletState:
WalletState(sequence=7, encrypted_wallet='czo4MTkyOjE2OjE6csXFcg5GaaIXauqORyfoSo3rcKiWzmTE2M/YBIJ1wmdBafqtUnKSO8DE/3EeKA35ow8iXJy5mowe4Ar8R+7m6FHxDblkDohjoeP9uZ5ziEirMSPu4eZsOcpXLdBsHp/qcGpNKAFnwGqeSdrxjvyFDQCyjl204mduE/X9mh6mlyYIei1IkK08rSTmc7mCuxIj')
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
WalletState(sequence=8, encrypted_wallet='czo4MTkyOjE2OjE6p5qwc7BWPjZE6sDI6To/dS5wVDMvzb6BA9oezrp3ecVryouLorggVTmnQLpcLdskBDNjE7/S5P4T4LT6hJCjAby2LCgeHuatDrySson8RHcs9rozLFoaPIQCDMkCPC8EGTN/g0aOAnDGvB6jaW/IsSLpMyXeth2OjloOtj2caT1hrihpThFrLPp4glBRLGL5')
'Success'
```

We generate a new salt seed when we change the password

```
>>> c1.salt_seed
'9968ede44e397d7875c057e4ebe50fd1118dc43a3e404e134353be6224947aad'
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
Got auth token:  796ea0575fe1ba5d6a43afec016f6ed2c9225a5180e76e744aad5b8857c8702b
>>> c2.get_auth_token()
Error 401
b'{"error":"Unauthorized: No match for email and password"}\n'
>>> c2.set_local_password("eggsandwich")
Generating keys...
Done generating keys
>>> c2.salt_seed
'9968ede44e397d7875c057e4ebe50fd1118dc43a3e404e134353be6224947aad'
>>> c2.get_auth_token()
Got auth token:  61932e198643b6f8bf1cd42fd0d296ce01b9813b6c1f312826baca3d50f95d47
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
WalletState(sequence=9, encrypted_wallet='czo4MTkyOjE2OjE6i9/lJoTH+8okCoczA7Q0o/X/X8MfVulO3qAq2GtKEKW9m4JH7Fcup62BOhwHtsNPOHIiMv9er5SpOx9pGBq3s9Bei4k2fNq4RXmXEPZX66p1T26VboJ0o53etIhnfQ9Q3pdLssiURjkK4OXpDDzbw1KsYXAPnJS50Nb8/A7+14BTgLoyJmGjW3nwjTxFjzRt')
'Success'
>>> c1.change_password("starboard")
Generating keys...
Done generating keys
Successfully updated password and wallet state on server
Synced walletState:
WalletState(sequence=10, encrypted_wallet='czo4MTkyOjE2OjE6D2TVSTbVugAqf8VSDmiIbArhQypf6o1LkQxAphJFlcbHHDtCQ4aqpmhsdWrkaf0Jd7+L2J12aWNf+XQGwS/kddTi9pplCwlAJB/RkTl4xeFPIRT5g5iVHrJcvCU2/gNa1BpRGtROn0fFb0SscS3WPxdQ5QA6jIGEblu1OY0KF/phpQt1DtNcuOS6rOz3NCmV')
'Success'
```
