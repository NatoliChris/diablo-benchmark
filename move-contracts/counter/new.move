script {
  use Owner::Counter;

  fun main(s: signer) {
    Counter::publish(&s, 0);
  }
}
