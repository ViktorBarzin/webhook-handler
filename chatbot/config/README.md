# RBAC config format

```yaml
---
permissions:
- idstr: "some-unique-permission-id"
  .
  .
  .

commands:
- id: "some-unique-command-id"
  cmd: "some shell command for the chatbot to execute"
  prettyName: "pretty name of the command"
  permissions:
    - "some-unique-permission-id"  # must refer an existing permission
  .
  .
  .
roles:
- id: "some-unique-role-id"
  permissions:
    - "some-unique-permissions-id" # must refer an existing permission
  .
  .
  .
groups:
- name: "some-unique-group-id"
  roles:
    - "some-unique-role-id" # must refer an existing role 
  .
  .
  .
users:
- id: "some-unique-user-id"
  name: "Pretty user name"
  roles:  # this is optional
  - "unique-role-id" # must refer an existing role 
  groups:  # this is optional
  - "some-unique-group-id" # must refer an existing group
  

```
Permissions are transitive. 
I.E if group `A` has permission `p` and user `U` has group `A`, then user `U` is has permission `p`.

# Chatbot conversation state machine config format
(TODO)

```yaml
statemachine:
- name: "GetStarted"
  src: 
    - "Initial"
  dst: "Your entry state ID"
```

Caveats which will be addressed at some point:
- Initial state is must have id "Initial" as that's what the FSM expects
- The "Get Started" button sends "GetStarted" as payload. This means your fsm should begin with:
