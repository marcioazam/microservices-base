package authz

# Attribute-based access control rules

# Allow access during business hours
allow {
    input.environment["hour"] >= 9
    input.environment["hour"] < 17
    input.subject.attributes["department"] == input.resource.attributes["department"]
}

# Allow access from trusted locations
allow {
    trusted_locations := {"office", "vpn", "datacenter"}
    trusted_locations[input.environment["location"]]
    input.subject.attributes["clearance_level"] >= input.resource.attributes["sensitivity_level"]
}

# Time-limited access
allow {
    input.subject.attributes["access_expires_at"] > input.environment["current_time"]
    input.action == "read"
}

# Project-based access
allow {
    input.subject.attributes["projects"][_] == input.resource.attributes["project_id"]
}
