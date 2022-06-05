module Owner::Stock {
  struct Stock has key { num: u64 }

  public fun new(account: &signer, num: u64) {
    move_to(account, Stock { num })
  }

  public fun buy(addr: address, qtt: u64) acquires Stock {
    let num_ref = &mut borrow_global_mut<Stock>(addr).num;

    if (*num_ref < qtt) abort 0;

    *num_ref = *num_ref - qtt;
  }
}
