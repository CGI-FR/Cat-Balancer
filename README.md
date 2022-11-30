# Cat Balancer

Cat Balancer is line based load balancer for net cat `nc`.

## Features

- Many producers
- Many consumers
- Close consumers socket on last producer close event
- Individual rate metrics in line / second for all peers
- Back Pressure detection for each producer
- Start a new session after drain consumers
- Optionaly wait for configured consumers and producers pool

## Usage

```
cb [-p <producers-port>] [-c <consumers-port>] [-i <interval>] [-C <consumers pool size>] [-P <producers pool size>]
```

###  One Producer to One Consumer

![One to One](doc/one_to_one.gif)

###  One Producer to Many Consumers

![doc/multi_consumers.gif](doc/multi_consumers.gif)

###  Many Producers to One Consumer

![doc/multi_producers.gif](doc/multi_producers.gif)

###  Many Producers to Many Consumers

![doc/multi_consumers_and_producers.gif](doc/multi_consumers_and_producers.gif)
