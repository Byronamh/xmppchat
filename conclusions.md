#Learnings from this project

## About Go

I had no previous experience with Go, so learning as I worked was an interesting challenge.
With every university project I try to make it a new challenging and rich experience. I could have developed this project in javascript or python in a day, since I'm familiar with the language, it's implementations and its syntax. Instead, I chose to learn a new technology and challenge myself to make a working project on unknown waters.

## Packages & dependencies
Go has its own package manager, encapsulating dependencies for a module in the `go.mod` file. 
You add new dependencies via the `go get <dep>` command. The `go.mod` file also includes the go version that's currently running.

## Language specifics

### General
Go is a heavily opinionated compiler, not allowing unused variables or imports. 
At one moment the compiler simply gave the message "too many errors". 
This surprised me, since it basically mean, fix the first few, and then we can go on.

### Channels
The most impressive feature (for me) was the channel (`chan`) type and the flexibility of its implementations.
Working with asynchronous events in an application that must share values or action upon "events" is something that I'm familiar with, with Rx library on javascript. 
So implementing something *similar* to observables was second nature to me. 

### Goroutines/threads
Go doesn't allow for thread creation explicitly, you can however call functions via the `go` method.
This will execute a function and work with it simultaneously to main function execution. This also brings interesting flexibility for working with asynchronous events, for example HTTP requests or the implementations done in this project.

### Strict implementation of typing and return values
As mentioned before, this is a strict language, forcing you to assign all vales returned by a function. 
For example, say a function returns the following: `(string, error)`, you **must** assign to variables that not only hold, but also handle said values. 

## Xmpp
This protocol is simple at is core and is even simpler with the abstractions given by the implemented library. 
It allows for many applications, not only chat based clients, but I also see potential with IoT devices. 
That being said, I wonder if implementing an XMPP based protocol is optimal against an HTTP/REST based client.

### Stanzas/IQ
I found the XML implementation for XMPP protocol interesting. 
Like mentioned above, it's simple at its core, but provides powerful flexibility when it comes to messaging other entities.

