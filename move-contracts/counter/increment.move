script {
  use Owner::Counter;

  fun main(a: address, count: u64) {
    Counter::increment(a, count);
  }
}
