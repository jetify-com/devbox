package mutagenbox

// mutagenbox is a package that encapsulates state and logic specific to how
// we need to manage mutagen for the devbox cloud.
//
// Also, resolves some compile cycles:
//   - [cloud] depends on [mutagenbox], [sshshim], and [mutagen].
//   - [sshshim] depends on [mutagenbox] and [mutagen].
