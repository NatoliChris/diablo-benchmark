module Owner::Counter {
  struct Counter has key { i: u64 }

  public fun publish(account: &signer, i: u64) {
    move_to(account, Counter { i })
  }

  public fun get(addr: address): u64 acquires Counter {
    borrow_global<Counter>(addr).i
  }

  public fun increment(addr: address, n: u64) acquires Counter {
    let c_ref = &mut borrow_global_mut<Counter>(addr).i;
    *c_ref = *c_ref + n
  }
}
