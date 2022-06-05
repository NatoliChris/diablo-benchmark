script {
  use Owner::Stock;

  fun main(s: signer) {
    Stock::new(&s, 10000000);
  }
}
