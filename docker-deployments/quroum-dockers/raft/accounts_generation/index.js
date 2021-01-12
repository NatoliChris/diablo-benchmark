const Wallet = require("ethereumjs-wallet")
const fs = require("fs")

let stream = fs.createWriteStream("accounts_raw", {flags: 'a'})


const ACCNUM = 660000

for (let i = 0; i < ACCNUM; i++) {
    a = Wallet.default.generate();
    stream.write(a.getAddressString() + ":" + a.getPrivateKeyString() + "\n")

    if (i % 2000 == 0) {
        console.log("[*] Progress: " + i + "/" + ACCNUM)
    }
}
