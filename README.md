# Cat Balancer

Cat Balancer is line based load balancer for net cat `nc`.

## Usage

```
cb [-p <producers-port>] [-c <consumers-port>]
```

###  One Producer to One Consumer

![One to One](doc/one_to_one.gif)

###  One Producer to Many Consumers

![doc/multi_consumers.gif](doc/multi_consumers.gif)

###  Many Producers to One Consumer

![doc/multi_producers.gif](doc/multi_producers.gif)

###  Many Producers to Many Consumers

![doc/multi_consumers_and_producers.gif](doc/multi_consumers_and_producers.gif)