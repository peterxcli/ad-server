# Dcard Backend Intern Homework 2024

## Overview

看到這個題目要求的時候, 就在想說這個 QPS > 10000 是單純可以用一個 redis 就可以解決的問題嗎?, 所以我就開始思考這個問題, 並且想到了一個比較有趣的解法, 這個解法是用一個 in-memory database 來處理這個問題, 並且用一個 redis stream 來處理 log ordering, 並且用 postgresql 來處理持久化的部分, 因為是 local 的 in-memory database, 所以只要透過像是 K8s Deployment 或是 `docker compose --scale` 就可以無限擴展讀取的操作的速度, 不過寫入的話就還是受限於 `max(redis, postgres)` 的速度, 我在實作裡已經盡力讓系統是 fault tolerance & consistency 的, 如果有人注意到我有哪些 case 沒有考慮到或是處理得不好可以優化的地方, 再麻煩各位提出來, 謝謝!

![alt text](./img/overview.png)

The main components in my system design idea have three parts, which can correspond to the `Servers` in the above figure respectively.

### Components

#### State Machine

For each instance, it is a state machine that can handle the advertisement CRUD operation and the range query operation. In the above diagram, it should use single-threaded to guarantee the read and write order. In Our Scenario, the consistency isn't the most important thing, so we can use `Readers–writer lock` to handle the concurrent read, the write operation is still single-threaded.

#### Consensus & Log Ordering

It is hard to implement a Linearizable Log System. so I can use `Redis Stream` to handle the log ordering and the log replication.

> Use redis lock to prevent the concurrent write to postgres and redis stream

#### Snapshot & Recovery

The state machine can be recovered from the snapshot, and the snapshot only modified if there is a new create, update, or delete operation. The snapshot can be stored in postgresql, and the recovery process can be done by the snapshot and the log to prevent the state machine need to replay all the log from the beginning. The concept is similar to the `AOF` and `RDB` in redis.

## Implement Practice

### Persistence Layer - PostgreSQL

- each advertisement is stored in the `ad` table, the multi-choice field is stored as string array(postgresql array type)

```go
type Ad struct {
 ID       uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
 Title    string         `gorm:"type:text" json:"title"`
 Content  string         `gorm:"type:text" json:"content"`
 StartAt  CustomTime     `gorm:"type:timestamp" json:"start_at" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
 EndAt    CustomTime     `gorm:"type:timestamp" json:"end_at" swaggertype:"string" format:"date" example:"2006-01-02 15:04:05"`
 AgeStart uint8          `gorm:"type:integer" json:"age_start"`
 AgeEnd   uint8          `gorm:"type:integer" json:"age_end"`
 Gender   pq.StringArray `gorm:"type:text[]" json:"gender"`
 Country  pq.StringArray `gorm:"type:text[]" json:"country"`
 Platform pq.StringArray `gorm:"type:text[]" json:"platform"`
 Version   int           `gorm:"index" json:"version"`
 IsActive  bool          `gorm:"type:boolean; default:true" json:"-" default:"true"`
 CreatedAt CustomTime    `gorm:"type:timestamp" json:"created_at"`
}
```

### Log Layer - Redis Stream

> no leader, no follower, all instance(replica) are equal

- use `XADD` to append the log (create, update, delete)
  - the publisher replica did not update its inmemory database at the same time
- all instance subscribe with `XREAD` to get the log
- the inmemory database for each replica only update if the replica receive the log from the redis stream

TODO: image from redis insight

### In-Memory Database (Local)

- multi-read/single-write lock
- implement the advertisement store by map with id primary key
- implement the advertisement indexing by map[string]mapset.Set[string]
  - By the way, originally I was using `map[string]map[string]*model.Ad`, and the concurrent read speed was only 4000 QPS. After changing it to `map[string]mapset.Set[string]`, the concurrent read speed increased to over 10000 QPS!!!
  - upd: I leverage the characteristic of `Pointer is Comparable` in Golang, then the performance become: write: 407676.68 QPS / read: 22486.06 QPS
  - I'm considering implementing multi-indexing to improve the read performance, not yet implemented currently
  - upd: I have tried to implement the multi-indexing, the write performance is down, but the read performance is now 1166960 QPS, so I think it's worth it - [commit detail](https://github.com/peterxcli/dcard-backend-2024/commit/028f68a2b1e770aac0754331826fd3110aa0b977)
- ~~implement the advertisement range query(ageStart, ageEnd, StartTime, EndTime) by interval tree~~
  - I have tried some interval tree library, but the read performance is not good, so I give up this implementation
  - Currently, I just iterate all the advertisement and filter the result by the condition

#### Benchmark

> if interval tree is in use, it doesn't apply on time range query since the performance issue

1. github.com/rdleal/intervalst
![alt text](./img/rdleal-interval-inmem.png)
2. github.com/biogo/store/interval
![alt text](./img/biogo-interval-inmem.png)
3. Just iterate all the advertisement and filter the result by the condition
![alt text](./img/iterate-inmem.png)

### Fault Recovery

- The recovery process is done by the snapshot and the log to prevent the state machine need to replay all the log from the beginning
- the snapshot only modified if there is a new create, update, or delete operation
- the snapshot can be stored in postgresql
- retry if the snapshot version and the log version is not match
- if there aren't any problem, start to subscribe the log from the snapshot version and replay the log

## Testing

### Unit Test

- gotests auto generate test functions
- [redis mock](https://github.com/go-redis/redismock/v9)
- [sqlmock](https://github.com/DATA-DOG/go-sqlmock)

### K6 Load Test

## Misc

### Test Coverage

<https://dcard-backend-intern-2024.peterxcli.dev/coverage>

### Swagger API Document

<https://dcard-backend-intern-2024.peterxcli.dev/docs>

### Code Statistic

![alt text](./img/gocolc.png)

<!-- ## 後話 -->

<!-- 1. 可以用 postgres CDC 來同步變動到 queue 裡面來達到更好的資料一致性, 但這只是一個 POC, 所以暫時沒有實作 -->

## Reference

- [sorted set](https://stackoverflow.com/a/32080338)
<!-- - [pglogical(pg cdc full row)](https://github.com/2ndQuadrant/pglogical) -->