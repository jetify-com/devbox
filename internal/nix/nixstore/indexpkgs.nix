#!/usr/bin/env nix eval --read-only --show-trace --json --file

/* indexpkgs.nix is an expression that starts with the top-level attributes in
   nixpkgs and recursively walks the tree looking for derivations. The output is
   an attribute set containing package information keyed by hash.

   This expression doesn't descend into attribute sets that are missing the
   recurseForDerivations attribute. Derivations that fail to evaluate are
   silently skipped.

   # Performance

   Evaluating all of nixpkgs is slow and consumes large amounts of memory. On a
   fast machine this expression takes about 40s to execute and requires up to
   20 GiB of RAM. Be aware of any performance impacts your changes may have.

   # Debugging

   Put the following shebang at the top of this file and make it executable in
   order to run it with `./indexpkgs.nix | jq .`:

     #!/usr/bin/env nix eval --read-only --show-trace --json --file
*/

with builtins;

let
  /* When changing this file, try to only use built-in Nix functions and avoid
     any functions or constants in nixpkgs itself. Otherwise this expression
     will fail when run against an older nixpkgs commit that doesn't have the
     necessary dependencies.
  */
  pkgs = import
    (fetchTarball {
      url = "https://github.com/nixos/nixpkgs/archive/{{ . }}.tar.gz";
    })
    # Comment out the fetchTarball above and uncomment the below function below
    # to debug with a local clone of nixpkgs.
    #
    # (fetchGit {
    #   url = "<path-to-repo>";
    #   rev = "3364b5b117f65fe1ce65a3cdd5612a078a3b31e3";
    #   allRefs = true;
    #   ref = "master";
    # })
    {
      config = {
        # Always include unfree or broken packages in the index. Devbox can choose
        # whether or not to show them in search results.
        allowUnfree = true;
        allowBroken = true;
      };
    };

  /* isDerivation returns true if x evaluates to a derivation.

     Type:
       isDerivation :: Any -> Bool
  */
  isDerivation = x: (x.type or null) == "derivation";

  /* shouldRecurse returns true if walkNixpkgs should descend into x to look for
     more derivations. It relies on a nixpkgs convention where a
     recurseForDerivations attribute is set to true when a set may contain
     child derivations.

     Type:
       shouldRecurse :: Any -> Bool
  */
  shouldRecurse = x: (tryEval (x.recurseForDerivations or false)).value;

  /* The following functions extract package information from various derivation
     attributes.
  */
  getName = drv: drv.pname or (parseDrvName drv.name).name;
  getVersion = drv: splitVersion (toString (drv.version or (parseDrvName drv.name).version));
  getHomepage = drv: if isList drv then head drv else drv;
  getLicense = drv: if isList drv then (head drv).spdxId else if isAttrs drv then drv.spdxId else drv;
  getPkgInfo = drv: {
    /* The name of the package without its version.

       Examples:
         go
         python3
         bash
    */
    name = getName drv;

    /* The package version split into a list of its components. The version is
       split using the splitVersion built-in function so that Devbox can do
       ordered comparisons.

       Examples:
         [ "1" "19" "3" ] # go 1.19.3
         [ "3" "11" "1" ] # python 3.11.1
    */
    version = getVersion drv;

    /* A list of attribute paths that point to this package. In other words,
       they all point to the same derivation with the same hash.

       Examples:
         [ "python3" "python310" "python310Packages.python" ... ]
         [ "go" "go_1_19" ]
    */
    paths = [ ];

    # The remaining attributes are all optional.
    ${if drv ? meta.mainProgram then "program" else null} = drv.meta.mainProgram;
    ${if drv ? meta.description then "summary" else null} = drv.meta.description;
    ${if drv ? meta.longDescription then "description" else null} = drv.meta.longDescription;
    ${if drv ? meta.homepage then "homepage" else null} = getHomepage drv.meta.homepage;
    ${if drv ? meta.license.spdxId then "license" else null} = getLicense drv.meta.license;
    ${if drv ? meta.platforms then "platforms" else null} = drv.meta.platforms;
    ${if drv ? meta.broken then "broken" else null} = drv.meta.broken;
    ${if drv ? meta.insecure then "insecure" else null} = drv.meta.insecure;
  };

  /* appendAttrPath adds an attribute path to a package info. A single package
     may have multiple paths pointing to it. For example, "python3"
     and "python310" will resolve to the same derivation if Python 3.10 is the
     default Python 3 interpreter.

     Type:
       appendAttrPath :: AttrSet -> [ String ] -> AttrSet
  */
  appendAttrPath = pkgInfo: attrPath: pkgInfo // {
    paths = pkgInfo.paths ++ [ (concatStringsSep "." attrPath) ];
  };

  /* walkNixpkgs starts at the top-level of nixpkgs and recursively walks its
     attributes looking for derivations, building up allPkgs as it goes. It
     returns an attribute set containing package info keyed by package hash.

     Each walkNixpkgs call enumerates the attributes in attrSet and attempts to
     evaluate their values as a derivation. If the evaluation succeeds, it adds
     the derivation's hash and package info to allPkgs. If the derivation's
     hash is already in allPkgs, it appends the current attribute path to the
     existing package info. Finally, it recursively calls itself on each
     attribute value (even if the attribute wasn't a derivation) to look for
     any derivations in nested attribute sets.

     Type:
       walkNixpkgs :: AttrSet -> [ String ] -> AttrSet -> AttrSet
  */
  walkNixpkgs = allPkgs: attrPath: attrSet: foldl'
    (allPkgs: attrName:
      let
        attrValue = attrSet."${attrName}";
        attrValuePath = attrPath ++ [ attrName ];
        tryDerivationHash = tryEval (
          if isDerivation attrValue then
            # unsafeDiscardStringContext allows the substring to be used as a
            # key in an attribute set. Nix reports an error otherwise.
            substring 0 32 (unsafeDiscardStringContext (baseNameOf attrValue.outPath))
          else
            null
        );
        derivationHash = if tryDerivationHash.success then tryDerivationHash.value else null;
        pkgInfo = appendAttrPath (allPkgs.${derivationHash} or (getPkgInfo attrValue)) attrValuePath;

        # Rely on the behavior where attributes are automatically omitted from a
        # set when their name is null. That makes this update a no-op when
        # derivationHash failed to evaluate or wasn't a derivation.
        updatedAllPkgs = allPkgs // { ${derivationHash} = pkgInfo; };
      in
      (
        if shouldRecurse attrValue then
          walkNixpkgs updatedAllPkgs attrValuePath attrValue
        else
          updatedAllPkgs
      )
    )
    allPkgs
    (attrNames attrSet);
in

  /* Keep the following things in mind if you're changing the JSON output:

       - Nix sorts JSON field names alphabetically. When renaming fields, make
         sure that package count comes before the array of packages so that
         Devbox can preallocate space.
       - Keep the JSON as flat as possible to simplify the Go parsing code and
         make debugging easier.
  */
rec {
  count = length (attrNames packages);
  system = currentSystem;
  nix = nixVersion;
  packages = walkNixpkgs { } [ ] pkgs;
}
