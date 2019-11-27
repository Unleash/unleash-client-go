/*
Package unleash is a client library for connecting to an Unleash feature toggle server.

See https://github.com/Unleash/unleash for more information.

Basics

The API is very simple. The main functions of interest are Initialize and IsEnabled. Calling Initialize
will create a default client and if a listener is supplied, it will start the sync loop. Internally the client
consists of two components. The first is the repository which runs in a separate Go routine and polls the
server to get the latest feature toggles. Once the feature toggles are fetched, they are stored by sending
the data to an instance of the Storage interface which is responsible for storing the data both in memory
and also persisting it somewhere. The second component is the metrics component which is responsible for tracking
how often features were queried and whether or not they were enabled. The metrics components also runs in a
separate Go routine and will occasionally upload the latest metrics to the Unleash server. The client struct
creates a set of channels that it passes to both of the above components and it uses those for communicating
asynchronously. It is important to ensure that these channels get regularly drained to avoid blocking those
Go routines. There are two ways this can be done.

Using the Listener Interfaces

The first and perhaps simplest way to "drive" the synchronization loop in the client is to provide a type
that implements one or more of the listener interfaces. There are 3 interfaces and you can choose which ones
you should implement:
 - ErrorListener
 - RepositoryListener
 - MetricsListener
If you are only interesting in tracking errors and warnings and don't care about any of the other signals,
then you only need to implement the ErrorListener and pass this instance to WithListener(). The DebugListener
shows an example of implementing all of the listeners in a single type.

Reading the channels directly

If you would prefer to have control over draining the channels yourself, then you must not call WithListener().
Instead, you should read all of the channels continuously inside a select. The WithInstance example shows how
to do this. Note that all channels must be drained, even if you are not interested in the result.

Examples

The following examples show how to use the client in different scenarios.

*/
package unleash
