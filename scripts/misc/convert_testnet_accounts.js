const fs = require('fs')

let f = fs.readFileSync("../accounts")
let parsed = JSON.parse(f)

let ob = []

for (let k in parsed["private_keys"]) {
    ob.push({
        "address": k,
        "private": "0x" + parsed["private_keys"][k]
    })
}

fs.writeFileSync('keys.json', JSON.stringify(ob))
