# ADR 045: TBD

## Changelog

- 21.05.2021: Initial Draft

# Authors
- Frojdi Dymylja (@fdymylja)

## Status

Draft

## Abstract

This ADR introduces a new sign mode which does not need message signers to know the fee-payer beforehand.

This ADR also extends the authentication information to include a `tip` which is a fee that fee payers can claim when they pay normal transaction fees.

## Context

Cosmos-sdk applications can hold a multitude of assets, which might be denoms created in the chain itself or denoms coming from other chains via IBC.

The problem is that they're forced to pay transaction fees by using denoms specified in the mempool fees. 

This becomes critical when it comes to enabling IBC's full potential as, for example, users moving assets from `chain B` to `chain A` are not able to transact in `chain A` unless they hold the coin used to pay for fees in `chain A`.

Buying the required asset to transact in a chain is a tedious operation as it requires users to register to an exchange, possibly go through a KYC process, deposit FIAT, buy coins, withdraw them to the address. This is a flow that adds a lot of friction when working via IBC and discourages people from experimenting with IBC.

Secondary solutions such as internal fee markets and all of that require a lot of research and implementations might take several months.

## Decision

To serve this use case we propose to introduce a new sign mode, called `SIGN_MODE_NAME_TBD`.

In order not to break client backwards client compatibility we're forced to work on data structures we can extend.

So we propose to extend AuthInfo in the following way:

`AuthInfo`:
```protobuf

```

This sig mode has two SignDocs called:

`SignDocIntent`:
```protobuf

```
`SignDocFees`:
```protobuf

```
## Consequences

### Backwards Compatibility

### Positive

### Negative


### Neutral


## Further Discussions

## References

