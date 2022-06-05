module Owner::Content {
  struct Content has key { data: vector<u8> }

  public fun publish(account: &signer, data: vector<u8>) {
    move_to(account, Content { data })
  }
}
