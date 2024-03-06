# Dcard Backend Intern Homework 2024

## Implement Practice

### Persistence Layer - PostgreSQL

### Log Layer - Redis Stream

### In-Memory Database (Local)

- multi-read/single-write lock
- implement the advertisement store by map with id primary key
- implement the advertisement indexing by map[string]mapset.Set[string]
  - By the way, originally I was using `map[string]map[string]*model.Ad`, and the concurrent read speed was only 4000 QPS. After changing it to `map[string]mapset.Set[string]`, the concurrent read speed increased to over 10000 QPS!!!
- ~~implement the advertisement range query(ageStart, ageEnd, StartTime, EndTime) by interval tree~~
  - I have tried some interval tree library, but the read performance is not good, so I give up this implementation
  - Currently, I just iterate all the advertisement and filter the result by the condition

#### Benchmark

> if interval tree is in use, it doesnt apply on time range query since the performance issue

1. github.com/rdleal/intervalst
![alt text](./img/rdleal-interval-inmem.png)
2. github.com/biogo/store/interval
![alt text](./img/biogo-interval-inmem.png)
3. Just iterate all the advertisement and filter the result by the condition
![alt text](./img/iterate-inmem.png)

### Recovery

## 後話

1. 可以用 postgres CDC 來同步變動到 queue 裡面來達到更好的資料一致性, 但這只是一個 POC, 所以暫時沒有實作
