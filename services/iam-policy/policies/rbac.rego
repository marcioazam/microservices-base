package authz

default allow = false

# Allow if user has admin role
allow {
    input.subject.attributes["role"] == "admin"
}

# Allow read access for users with read permission
allow {
    input.action == "read"
    input.subject.attributes["permissions"][_] == "read"
}

# Allow write access for users with write permission
allow {
    input.action == "write"
    input.subject.attributes["permissions"][_] == "write"
}

# Allow users to access their own resources
allow {
    input.subject.id == input.resource.attributes["owner_id"]
}

# Role-based access control
allow {
    required_role := role_permissions[input.resource.type][input.action]
    user_roles := input.subject.attributes["roles"]
    user_roles[_] == required_role
}

role_permissions = {
    "document": {
        "read": "viewer",
        "write": "editor",
        "delete": "admin"
    },
    "user": {
        "read": "admin",
        "write": "admin",
        "delete": "super_admin"
    }
}
