# Major Features

A major feature is a user-facing capability that represents a distinct real-world concern the software addresses.

* User-facing scope: Something an end user would recognize as a distinct thing the app does — for example, "send a message", "manage users", "process payments", "handle authentication"
* Bounded by domain, not infrastructure: It's defined by what the software does, not how — so "database access" or "HTTP routing" are not major features, but "message posting" or "channel management" are
* Covered by one or a small set of domain functions: A major feature should have identifiable entry points (handlers, core logic loops, event handlers) that summarize its behavior at a high level
* Independent enough to name: If you can describe it as a noun phrase a non-technical stakeholder would understand, it's likely a major feature