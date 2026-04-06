package main

// Bridge file — delegates to internal/config.
// Will be removed when all code moves to internal/.

import "github.com/Gaetan-Jaminon/fleetdesk/internal/config"

var parseFleetFile = config.ParseFleetFile
