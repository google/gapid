This is a highlevel overview of behaviors used by <bot-list>.

```
                        +----------------+
                        |                |
         +------------->+ CommonBehavior +<-----------------+
         |              |                |                  |
         |              +------+---------+                  |
         |                     ^                            |
         |                     |                            |
         |                     |                            |
         |                     |                            |
+--------+----------+     +----+------------+       +-------+--------------+
|                   |     |                 |       |                      |
| QueryColumnFilter |     | BotListBehavior |       | DynamicTableBehavior |
|                   |     |                 |       |                      |
+-------------------+     +-----------------+       +----------------------+
         ^  +-------------^ ^      ^ ^-------------------+     ^
         |  |               |      |                     |     |
         |  |               |      +---+                 |     |
         |  |               |          |                 |     |
    +----+--+-------+       |    +-----+--------------+  +-----+------+
    |               |       |    |                    |  |            |
    | <bot-filters> |       |    | <bot-list-summary> |  | <bot-list> |
    |               |       |    |                    |  |            |
    +---------------+       |    +--------------------+  +------------+
                            |     +-----------------+
                            |     |                 |
                            +-----+ <bot-list-data> |
                                  |                 |
                                  +-----------------+
```

Although methods defined in the various behaviors could be overwritten by each
other, this is not done.

`<bot-list>` is the main element and composes the others.