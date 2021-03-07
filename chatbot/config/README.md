Config format:

```yaml
---
fsm:
- name: "SomeEventID"  # MUST point to an existing event id
  srcState:
    - "Some source state"  # MUST point to existing state ID
    - "Another source state" # Same ...
  destState: "Some destination state" # Same
- name: ....

---
states:
- id: "SomeStateID"  # Used in `fsm` object
  message: "Message show to user at this state"
- id: ...

---
events:
- id: "SomeEventID"  # Used in `fsm` object
  message: "Message shown to user in form of a postback button"
  orderID: 10 # When multiple transitions are available, this is their explicit order (lower means higher pri)
- id: ...
```

Caveats which will be addressed at some point:
- Order of definition must be as shown above i.e 1.fsm, 2.states 3.events
- Initial state is must have id "Initial" as that's what the FSM expects
- The "Get Started" button sends "GetStarted" as payload. This means your fsm should begin with:

```yaml
fsm:
- name: "GetStarted"
  srcState: 
    - "Initial"
  destState: "Your entry state ID"
```
