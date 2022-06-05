script {
  use Owner::Stocks;

  fun main(a: address, idx: u64) {
    Stocks::buy(a, idx, 1);
  }
}
