script {
  use Owner::Match;

  fun main(s: signer) {
    Match::new(&s, 10000, 10000);
  }
}
