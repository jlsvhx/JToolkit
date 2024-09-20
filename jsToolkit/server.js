var http = require('http');
var events = require('events');
var eventEmitter = require('events').EventEmitter;
require('./hello')

var event = new eventEmitter();
event.on("launch", function() {
    console.log("Browser is launched");
})

console.log(global)