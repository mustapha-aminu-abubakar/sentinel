# Product Requirements Document (PRD)

# Project: Sentinel

## Global Rate Limiter as a Service

**Version:** 1.0
**Status:** MVP
**Document Type:** Product Requirements Document

---

# 1. Overview

## 1.1 Product Summary

Sentinel is a distributed, high-availability rate limiting platform that enables organizations to centrally manage API request quotas across multiple services and infrastructure instances.

Instead of allowing every microservice to independently enforce API limits, Sentinel provides a shared decision-making layer that determines whether outbound API requests should proceed.

The platform prevents:

* accidental API quota exhaustion
* unexpected third-party API billing
* inconsistent rate enforcement across service replicas
* duplicated rate-limit logic across applications

---

# 2. Problem Statement

Modern companies integrate with hundreds of external APIs:

* banking providers
* logistics providers
* AI providers
* payment gateways
* government APIs

These providers commonly enforce strict quotas.

Current architecture problems:

1. Each microservice maintains its own rate limits.
2. Multiple replicas are unaware of each other's usage.
3. The same quota is consumed multiple times due to distributed deployments.
4. Teams duplicate rate-limiting implementations.
5. Monitoring API consumption requires manual aggregation.

Example:

A company has:

```
Payment Service
 ├── Instance A
 ├── Instance B
 ├── Instance C
```

API provider limit:

```
1000 requests/minute
```

Without centralized control:

```
Instance A thinks:
"I have 1000 requests available"

Instance B thinks:
"I have 1000 requests available"

Instance C thinks:
"I have 1000 requests available"
```

Actual traffic:

```
3000 requests/minute
```

Result:

```
429 Too Many Requests
Unexpected charges
Service degradation
```

---

# 3. Goals

## Primary Goals

### 3.1 Distributed Rate Enforcement

Ensure all service instances share a consistent rate-limit state.

Requirements:

* Multiple Sentinel instances can run simultaneously.
* Any instance can process a request.
* Rate decisions remain globally accurate.

---

### 3.2 Ultra-Fast Rate Checks

The rate-limit decision path must complete within milliseconds.

Target:

```
< 5ms average latency
```

The request authorization path must prioritize speed over analytics.

---

### 3.3 High Availability

Temporary failures must not block all API traffic.

The system must tolerate:

* Redis downtime
* PostgreSQL downtime
* network interruptions

---

### 3.4 Usage Analytics

Every approved request must be recorded for:

* monitoring
* billing
* auditing
* optimization

---

### 3.5 Client Self-Service Dashboard

Clients should be able to:

* configure limits
* view usage
* analyze trends
* identify high-consuming APIs

---

# 4. Non Goals

The MVP will not include:

* API key generation
* billing/payment processing
* custom rate limiting algorithms
* API gateway replacement
* request transformation
* authentication management

---

# 5. Users

## 5.1 Platform Administrator

Responsible for:

* managing clients
* configuring limits
* monitoring system health

## 5.2 API Client

External/internal service consuming Sentinel.

Needs:

* request authorization
* usage visibility
* quota information

## 5.3 Engineering Team

Needs:

* deployment simplicity
* observability
* reliability

---

# 6. Functional Requirements

---

# 6.1 Client Management

## Description

The system must support multiple API consumers.

Example:

```
Client A
Limit:
100 requests/minute


Client B
Limit:
5000 requests/minute
```

---

## Features

Admin can:

* create clients
* update clients
* deactivate clients
* view usage

Client entity:

```
Client
---------
id
name
status
created_at
updated_at
```

---

# 6.2 Rate Limit Configuration

Admins can configure:

* request limit
* time window
* API identifier

Example:

```
Client:
OpenAI Integration

Limit:
5000 requests

Window:
1 minute
```

Database:

```
RateLimitRule

id
client_id
requests_allowed
window_seconds
created_at
updated_at
```

---

# 6.3 Rate Limit Check API

## Endpoint

```
POST /v1/check
```

Request:

```json
{
  "client_id": "client_123",
  "api": "openai"
}
```

Response:

Allowed:

```json
{
  "allowed": true,
  "remaining": 499
}
```

Rejected:

```json
{
  "allowed": false,
  "retry_after": 25
}
```

---

# 6.4 Rate Limiting Algorithm

## Algorithm

Sliding Window Counter

Reason:

* accurate compared to fixed windows
* simpler than token bucket
* Redis supports efficient implementation

Example:

Limit:

```
100 requests/minute
```

Request timeline:

```
10:00:00 50 requests
10:00:30 30 requests
10:01:00 remove expired requests
```

Current usage:

```
80 requests
```

Decision:

```
Allow
```

---

# 6.5 Redis Fast Path

Redis is the primary decision store.

Responsibilities:

* maintain active counters
* perform atomic increments
* store cached configuration

Flow:

```
Request
 |
 v
Sentinel
 |
 v
Redis
 |
 +---- Allowed
 |
 +---- Rejected
```

Redis operations:

```
INCR
EXPIRE
ZADD
ZRANGE
```

---

# 6.6 PostgreSQL Source of Truth

PostgreSQL stores:

* clients
* rules
* historical usage
* audit logs

Purpose:

* persistence
* reporting
* recovery

---

# 6.7 Cache Strategy

Configuration cache:

```
Redis

Client limits
API configuration
```

Database:

```
PostgreSQL
```

Flow:

```
Request
 |
Redis lookup
 |
Found?
 |
Yes --> Continue
 |
No
 |
Postgres
 |
Update Redis
```

---

# 7. Failure Handling Strategy

Requirement:

Database/cache failure must not make the service unavailable. 

---

# 7.1 Redis Failure

Scenario:

```
Redis unavailable
```

Behavior:

1. Attempt Redis connection.
2. If unavailable:

   * fallback to PostgreSQL.
3. If PostgreSQL unavailable:

   * enter fail-open mode.

Fail-open:

```
Allow request
Log degraded state
Alert monitoring
```

Reason:

The primary business goal is preventing unnecessary outages.

---

# 7.2 PostgreSQL Failure

Scenario:

```
Postgres unavailable
```

Behavior:

* Existing Redis state continues working.
* Configuration cached in Redis remains valid.
* Analytics writes are queued.

---

# 7.3 Analytics Failure

Analytics must never block authorization.

Flow:

```
Request approved

      |
      v

Async event queue

      |
      v

Analytics storage
```

---

# 8. Analytics System

## Requirements

Every approved request is logged. 

Events:

```json
{
 "client_id":"123",
 "api":"openai",
 "timestamp":"...",
 "latency":120,
 "status":"allowed"
}
```

---

# 9. Dashboard Requirements

## Dashboard Pages

---

# 9.1 Overview Dashboard

Metrics:

* total requests
* allowed requests
* rejected requests
* success percentage
* average latency

Example:

```
Requests Today

1,245,000


Blocked

2,430


Average Latency

8ms
```

---

# 9.2 Usage Analytics

Charts:

## Request Volume

Time ranges:

* 10 days
* 15 days
* 30 days

## Latency Trends

Graph:

```
Average Response Time

Day 1
Day 2
Day 3
```

---

## Filters

Support:

* client
* API provider
* date range
* status

---

# 9.3 Limit Management

Admin can:

Create:

```
Client:
Stripe Integration

Limit:
1000/min
```

Update:

```
5000/min
```

Disable:

```
Inactive
```

---

# 10. System Architecture

## High Level Architecture

```
                 Clients

                    |
                    |

             Sentinel API

                    |

        ----------------------

        |                    |

      Redis             PostgreSQL

        |                    |

 Rate Decisions       Persistent Data


                    |

             Analytics Queue

                    |

             Analytics DB

```

---

# 11. Proposed Technology Stack

## Backend

Option:

Go

Reason:

* excellent concurrency
* low latency
* efficient networking
* suitable for distributed systems

Framework:

```
Gin / Fiber
```

---

## Database

PostgreSQL

Purpose:

* persistent storage
* analytics queries

---

## Cache

Redis

Purpose:

* sliding window counters
* configuration caching

---

## Messaging

Optional:

Redis Streams

or

Kafka

Purpose:

* asynchronous analytics processing

---

## Frontend

Next.js

UI:

* Tailwind CSS
* shadcn/ui
* Recharts

---

# 12. Database Schema

## Client

```
clients

id
name
status
created_at
updated_at
```

---

## Rate Rules

```
rate_rules

id
client_id
limit
window_seconds
created_at
```

---

## Usage Logs

```
usage_logs

id
client_id
api
allowed
latency
created_at
```

---

# 13. API Specification

## Check Limit

```
POST /v1/check
```

---

## Clients

```
GET /clients

POST /clients

PATCH /clients/:id
```

---

## Rules

```
GET /rules

POST /rules

PATCH /rules/:id
```

---

## Analytics

```
GET /analytics/usage

GET /analytics/latency
```

---

# 14. Testing Requirements

The challenge specifically requires unit tests, race-condition tests, and load/performance tests. 

---

## Unit Tests

Test:

* limit calculation
* window expiration
* client isolation
* rule updates

---

## Race Condition Tests

Validate:

* concurrent requests
* atomic Redis operations
* multiple service instances

Example:

```
1000 concurrent requests

Expected:

Only 100 allowed
```

---

## Load Tests

Using:

* k6

Metrics:

* requests/sec
* latency
* error rate

Target:

```
10,000 checks/sec
```

---

# 15. Deployment Requirements

The challenge requires Docker configuration allowing the entire application stack to start using a single command. 

---

Docker Compose:

Services:

```
sentinel-api

postgres

redis

dashboard

analytics-worker
```

Command:

```bash
docker compose up --build
```

---

# 16. Observability

Metrics:

* request latency
* Redis availability
* database health
* rate-limit decisions
* failure mode activations

Tools:

* Prometheus
* Grafana
* OpenTelemetry

---

# 17. Success Metrics

MVP success:

| Metric                    | Target  |
| ------------------------- | ------- |
| Rate decision latency     | <5ms    |
| Availability              | 99.9%   |
| Concurrent requests       | 10k/sec |
| Incorrect limit decisions | 0       |
| Dashboard latency         | <2s     |

---

# 18. Deliverables

Final submission:

```
sentinel.zip

├── backend
├── dashboard
├── docker-compose.yml
├── Dockerfile
├── architecture.png
├── tests
├── README.md
```

The README must explain Docker execution, test execution, and verification of edge cases. 