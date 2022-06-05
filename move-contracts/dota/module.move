module Owner::Game {
  use Std::Vector;

  struct Game has key {
    max: u64,
    xs: vector<u64>,
    ys: vector<u64>
  }

  fun pow(base: u64, exp: u64): u64 {
    let res = 1;

    while (exp > 1) {
      res = res * base;
      exp = exp - 1;
    };

    res
  }

  public fun new(account: &signer, numplayers: u64, max: u64) {
    let xs = Vector::empty<u64>();
    let ys = Vector::empty<u64>();
    let i = 0;

    while (i < numplayers) {
      Vector::push_back(&mut xs, pow(i, 10) % max);
      Vector::push_back(&mut ys, pow(i + 1, 10) % max);
      i = i + 1
    };

    move_to(account, Game { max, xs, ys })
  }

  public fun update(addr: address) acquires Game {
    let game_ref = borrow_global_mut<Game>(addr);
    let i = 0;

    while (i < Vector::length<u64>(&game_ref.xs)) {
      let x = Vector::borrow_mut(&mut game_ref.xs, i);
      let y = Vector::borrow_mut(&mut game_ref.ys, i);

      *x = *x + 1;
      *y = *y + 1;

      if (*x > game_ref.max) {
        *x = 0
      };

      if (*y > game_ref.max) {
        *y = 0
      };

      i = i + 1
    }
  }
}
