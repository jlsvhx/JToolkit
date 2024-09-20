//hello.js
function Hello() {
    var name;
    this.setName = function(thyName) {
        name = thyName;
    };
    this.sayHello = function() {
        console.log('Hello ' + name);
    };
}
function getHello() {
    let tmp = new Hello();
    tmp.setName('World');
    return tmp;
}

var g1 = "1111";

exports.getHello = getHello;
// module.exports = Hello;