# Coordinate Catalog Rules in an Application Service

Category, Brand, Product, and Media remain modules within one Catalog context. Cross-module policies and transactions are coordinated by a Catalog application service over narrow module interfaces, rather than by concrete repositories querying other modules' tables, database triggers, or internal HTTP calls; this keeps catalog rules explicit and testable without splitting the current service.
