script {
  use Owner::Match;

  fun main(a: address) {
    Match::find(a);
  }
}
