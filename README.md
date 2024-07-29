# Bee Web Framework
Bee is a golang toy web framework for building web applications, written by Go.
Bee's core features include:

- [x] Routing.
- [x] Templates.
- [x] Middleware.
- [x] Recover.
- [ ] Utilities.

`updating and perfecting...`

# BeeCache
BeeCache is a mini implement of groupCache.It implements the core features of the distribute cache, including:

- [x] Use LRU algorithm to disuse the least visited record.
- [x] Single-machine concurrent cache.
- [x] Http Server.
- [x] Consistent hashing algorithm.
- [x] Distributed cache node.
- [x] SingleFlight.
- [x] Use Protobuf in the communication between the node.

# BeeORM
BeeORM is a mini implement of [xorm](https://xorm.io/)(And learn some code from gorm). It implements the core features of the ORM, including:
- [x] Logger.
- [x] Creation, Deletion of the table.
- [x] Primary key.
- [x] Crud operation of the record.
- [x] Hooks.
- [x] Transaction.
- [x] Migration of the table.

# BeeRPC
BeeRPC is a mini implement of rpc library of std lib `net/rpc`. It implements the core features of the RPC, including:
- [x] Protocol Exchange.
- [x] Registry.
- [x] Timeout Processing.
- [x] Service Discovery.
- [x] Load Balance.
- [ ] More Features.