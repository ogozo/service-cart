# Cart Service

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A microservice responsible for managing user shopping carts within the E-Commerce Microservice Platform.

## 📖 Overview

This service provides functionality for all shopping cart operations. It is designed for high performance and scalability, handling frequent reads and writes as users add, remove, and view items in their carts. Key responsibilities include:

*   **Cart Management:** Creating and managing persistent shopping carts for authenticated users.
*   **Item Operations:** Adding items to a user's cart and updating quantities.
*   **Saga Participation:** Listening for `OrderConfirmed` events to automatically clear a user's cart after a successful checkout.

This service uses Couchbase as its primary data store to leverage its flexibility and performance for session-like data.

## 🚀 Architecture

The Cart Service follows a Clean Architecture pattern (Handler, Service, Repository). It exposes a gRPC API for synchronous operations and includes a RabbitMQ consumer to react to events from other services.

```mermaid
graph TD
    subgraph "Request Flow"
        GW[🌐 API Gateway] -- gRPC --> Handler;
        Handler -- Calls --> BusinessService[Service];
        BusinessService -- Uses --> Repository;
        Repository -- CRUD --> DB[(🧩 Couchbase)];
    end
    
    subgraph "Event Flow (Saga)"
        RMQin[("📨 RabbitMQ")] -- "OrderConfirmed Event" --> Consumer;
        Consumer -- Calls --> BusinessService;
        BusinessService -- "Clear Cart" --> Repository;
    end

    subgraph "Observability"
        Handler -- "Traces/Logs/Metrics" --> OBS[(🔭 Observability Stack)];
        Repository -- "Traces/Logs" --> OBS;
        Consumer -- "Traces/Logs" --> OBS;
    end
```

### Why Couchbase?

A NoSQL document database like Couchbase was chosen for this service due to the nature of shopping cart data:
*   **Flexible Schema:** A user's cart can be easily represented as a single JSON document, which can evolve without rigid schema migrations.
*   **High Performance:** Carts are subject to frequent updates. Couchbase's memory-first architecture provides low-latency reads and writes, ideal for this use case.
*   **Scalability:** Couchbase can be easily scaled horizontally to handle a large number of concurrent users.

## 🛠️ Tech Stack

| Category | Technology |
| :--- | :--- |
| **Language** | Go |
| **Communication** | gRPC (Server), RabbitMQ (Consumer), Protobuf |
| **Database** | Couchbase (`gocb/v2`) |
| **Observability** | OpenTelemetry (`otelgrpc`, `gocb-opentelemetry`), Prometheus, Zap |
| **Containerization** | Docker |

## ⚙️ Setup & Running

This service is designed to be run as part of the complete platform using Docker Compose.

### Prerequisites

*   Docker & Docker Compose
*   All project repositories must be cloned at the same directory level.

### Running

The service is automatically built and started when you run the main `docker-compose up` command from the `docker-compose-environment` repository. The platform includes an `init-couchbase` container that automatically creates the required `ecommerce` bucket.

For detailed instructions, please see the [main project README](https://github.com/ogozo/docker-compose-environment/blob/main/README.md).

### Environment Variables

The service is configured via the `env/cart.env` file in the main compose directory.

| Variable | Description | Example |
| :--- | :--- | :--- |
| `GRPC_PORT` | The port on which the gRPC server listens. | `:50053` |
| `METRICS_PORT`| The port for the `/metrics` HTTP endpoint. | `:9003` |
| `COUCHBASE_CONN_STR`| Connection string for the Couchbase cluster. | `couchbase://couchbase-db`|
| `COUCHBASE_USER`| Username for Couchbase authentication. | `admin` |
| `COUCHBASE_PASS`| Password for Couchbase authentication. | `secret` |
| `COUCHBASE_BUCKET`| The Couchbase bucket used for storing carts. | `ecommerce` |
| `RABBITMQ_URL`| Connection string for the RabbitMQ broker. | `amqp://guest:guest@rabbitmq:5672/` |
| `OTEL_SERVICE_NAME`| Service name for observability. | `service-cart` |
| `OTEL_EXPORTER_OTLP_ENDPOINT`| OTLP endpoint for Jaeger. | `jaeger-tracer:4317` |

## 📡 gRPC & Event API

This service exposes its functionality via the `CartService` gRPC interface and listens for RabbitMQ events.

### gRPC API

*   **`rpc GetCart(GetCartRequest) returns (GetCartResponse)`**: Retrieves the current state of a user's shopping cart.
*   **`rpc AddItemToCart(AddItemToCartRequest) returns (AddItemToCartResponse)`**: Adds a specified quantity of a product to the user's cart. If the product already exists, the quantity is updated.
*   **`rpc ClearCart(ClearCartRequest) returns (ClearCartResponse)`**: Removes all items from a user's cart.

### Asynchronous API (Events)

*   **Listens for:** `OrderConfirmed` Event
    *   **Action:** When this event is received, the service permanently deletes the corresponding user's cart document from Couchbase, completing the checkout flow.
