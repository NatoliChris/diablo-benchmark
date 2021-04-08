const Wallet = require("ethereumjs-wallet")
const fs = require("fs")

let stream = fs.createWriteStream("accounts_raw", {flags: 'a'})

const spinnerframes = ['\\', '|', '-', '/', '-', '|']

const ACCNUM = 1000000
let sf = 0

for (let i = 0; i < ACCNUM; i++) {
    a = Wallet.default.generate();
    stream.write(a.getAddressString() + ":" + a.getPrivateKeyString() + "\n")

    if (i % 2000 == 0) {
        // console.log("[*] Progress: " + i + "/" + ACCNUM)
        process.stdout.write(`\r[${spinnerframes[sf++]}] Progress: ${i} / ${ACCNUM}`)
        sf %= spinnerframes.length
    }

}

process.stdout.write("\n[*] DONE")

