const Wallet = require('ethereumjs-wallet')
const fs = require('fs')

let stream = fs.createWriteStream('accounts.json', {flags: 'a'})

stream.write("[\n")

const ACCNUM = 220000

for (let i = 0; i < ACCNUM; i++) {
    a = Wallet.default.generate();
    stream.write("{\"address\": \"" + a.getAddressString() + "\", \"private\": \"" + a.getPrivateKeyString() + "\"}")

    if (i == ACCNUM-1) {
        stream.write("\n")
    } else {
        stream.write(",\n")
    }

    if (i % 2000 == 0) {
        console.log("Progress: " + i)
    }
}

stream.write("]")
stream.end()
