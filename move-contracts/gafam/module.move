module Owner::Stocks {
  use Std::Vector;

  struct Stocks has key { nums: vector<u64> }

  public fun new(account: &signer, nums: vector<u64>) {
    move_to(account, Stocks { nums })
  }

  public fun buy(addr: address, idx: u64, qtt: u64) acquires Stocks {
    let nums_ref = &mut borrow_global_mut<Stocks>(addr).nums;
    let num_ref = Vector::borrow_mut(nums_ref, idx);

    if (*num_ref < qtt) abort 0;	  

    *num_ref = *num_ref - qtt;
  }
}
