script {
  use Std::Vector;
  use Owner::Stocks;

  fun main(s: signer) {
    let stocks = Vector::empty();

    Vector::push_back(&mut stocks, 10000000u64);
    Vector::push_back(&mut stocks, 10000000u64);
    Vector::push_back(&mut stocks, 10000000u64);
    Vector::push_back(&mut stocks, 10000000u64);
    Vector::push_back(&mut stocks, 10000000u64);

    Stocks::new(&s, stocks);
  }
}
