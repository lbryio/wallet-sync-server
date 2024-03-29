This software is in the pre-testing phase as of now.

# Getting The Binary

## Prebuilt

Linux Only. Get the latest [release](https://github.com/lbryio/wallet-sync-server/releases).

## Building

Only tried on Linux. Might work for Windows and Mac. No expectations.

Install Golang, at least version 1.18. (Please report any dependencies we seemed to have forgotten)

Check out the repo and run:

```
go build .
```

The binary should show up as `wallet-sync-server`.

# Account Creation Settings

When running the server, we should set some environmental variables. These environmental variables determine how account creation is handled. If we do not set these, no users will be able to create an account.

## `ACCOUNT_VERIFICATION_MODE`

The allowed values are `AllowAll`, `Whitelist`, and `EmailVerify`.

### `ACCOUNT_VERIFICATION_MODE=AllowAll`

This should _only be used for development_. Unless you really just want anybody creating accounts on your server, hypothetically DOSing you, etc etc. This puts no restrictions on who can create an account and no process beyond simply pushing the "sign up button" (i.e. sending the "signup" request to the server).

### `ACCOUNT_VERIFICATION_MODE=Whitelist` (default)

With this option, only specifically whitelisted email addresses will be able to create an account. This is recommended for people who are self-hosting their wallet sync server for themself or maybe a few friends.

With this option, we should also specify the whitelist.

#### `ACCOUNT_WHITELIST`

This should be a comma separated list of email addresses with no spaces.

**NOTE**: If your email address has weird characters, unicode, what have you, don't forget to bash-escape it properly.

Single address example:

```
ACCOUNT_WHITELIST=alice@example.com
```

Multiple address example:

```
ACCOUNT_WHITELIST=alice@example.com,bob@example.com,satoshi@example.com
```

_Side note: Since `Whitelist` it is the default value for `ACCOUNT_VERIFICATION_MODE`, and since `ACCOUNT_WHITELIST` is empty by default, the server will by default have an empty whitelist, thus allowing nobody to create an account. This is the default because it's the safest (albeit most useless) configuration._

### `ACCOUNT_VERIFICATION_MODE=EmailVerify`

With this option, you need an account with [Mailgun](mailgun.com). Once registered, you'll end up setting up a domain (including adding DNS records), and getting a private API key. You'll also be able to use a "sandbox" domain just to check that the Mailgun configuration otherwise works before going through the process of setting up your real domain.

With this mode, we require the following additional settings:

#### `MAILGUN_SENDING_DOMAIN`

The address in the "from" field of your registration emails. Your Mailgun sandbox domain works here for testing.

#### `MAILGUN_SERVER_DOMAIN`

The server domain will determine what domain is used for the hyperlink you get in your registration confirmation email. You should generally put the domain you're using to host your wallet sync server.

Realistically, both sending and server domains will often end up being the same thing.

#### `MAILGUN_PRIVATE_API_KEY`

You'll get this in your Mailgun dashboard.

#### `MAILGUN_DOMAIN_IS_EU` (optional)

Whether your sending domain is in the EU. This is related to GDPR stuff I think. Valid values are `true` or `false`, defaulting to `false`.

# Deployment

A setup that works is [Caddy server](https://caddyserver.com) and Systemd.

Make sure Caddy is set to port 443, because the LBRY clients will expect that.

If you're using Mailgun, take care to keep the environmental vars secure. [See here](https://serverfault.com/questions/413397/how-to-set-environment-variable-in-systemd-service/910655#910655) for how to do this with systemd.
