Config format:

```yaml
---
statemachine:
- name: "SomeEventID"  # MUST point to an existing event id
  src:
    - "Some source state"  # MUST point to existing state ID
    - "Another source state" # Same ...
  dst: "Some destination state" # Same
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
- Initial state is must have id "Initial" as that's what the FSM expects
- The "Get Started" button sends "GetStarted" as payload. This means your fsm should begin with:

```yaml
statemachine:
- name: "GetStarted"
  src: 
    - "Initial"
  dst: "Your entry state ID"
```
