module Owner::Match {
  use Std::Hash;
  use Std::Vector;

  struct Match has key {
    max: u64,
    driver_xs: vector<u64>,
    driver_ys: vector<u64>,
    client_x: u64,
    client_y: u64
  }

  public fun new(account: &signer, numdrivers: u64, max: u64) {
    let h = Hash::sha3_256(Vector::empty<u8>());
    let xs = Vector::empty<u64>();
    let ys = Vector::empty<u64>();
    let max128 = (max as u128);
    let seed: u128 = 0;
    let i = 0;

    while (i < 8) {
      seed = (seed << 8) + (*Vector::borrow(&h, i) as u128);
      i = i + 1
    };

    i = 0;

    while (i < numdrivers) {
      let x = ((((i as u128) * seed) % max128) as u64);
      let y = (((((i + 1) as u128) * seed) % max128) as u64);
      Vector::push_back(&mut xs, x);
      Vector::push_back(&mut ys, y);
      i = i + 1
    };

    let x = ((((numdrivers as u128) * seed) % max128) as u64);
    let y = (((((numdrivers as u128) + 1) * seed) % max128) as u64);

    move_to(account, Match {
      max: max,
      driver_xs: xs,
      driver_ys: ys,
      client_x: x,
      client_y: y
    })
  }

  fun sqrt(n: u64): u64 {
    let z = (n + 1) / 2;
    let y = n;

    while (z < y) {
      y = z;
      z = (n / z + z) / 2;
    };

    z
  }

  public fun find(addr: address): u64 acquires Match {
    let match = borrow_global_mut<Match>(addr);
    let cx = match.client_x;
    let cy = match.client_y;
    let min_dist = 0;
    let min = 0;
    let i = 0;

    while (i < Vector::length<u64>(&match.driver_xs)) {
      let dx = *Vector::borrow(&match.driver_xs, i);
      dx = if (dx >= cx) dx - cx else cx - dx;
      dx = dx * dx;

      let dy = *Vector::borrow(&match.driver_ys, i);
      dy = if (dy >= cy) dy - cy else cy - dy;
      dy = dy * dy;

      let d = sqrt(dx + dy);

      if (i == 0) {
        min_dist = d;
      } else if (d < min_dist) {
        min = i;
	min_dist = d;
      };

      i = i + 1
    };

    min
  }
}
