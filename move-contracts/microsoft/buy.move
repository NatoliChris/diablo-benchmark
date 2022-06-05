script {
  use Owner::Stock;

  fun main(a: address) {
    Stock::buy(a, 1);
  }
}
