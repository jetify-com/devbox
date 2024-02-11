const std = @import("std");

pub fn build(b: *std.Build) void {
    // Standard target options allows the person running zig build to choose what
    // target to build for. By default, any target is allowed, and no choice means to
    // target the host system. Other options for restricting supported target set are
    // available.
    const target = b.standardTargetOptions(.{});

    // Standard optimization options allow the person running zig build to select
    // between Debug, ReleaseSafe, ReleaseFast, and ReleaseSmall. By default none of
    // the release options are considered the preferable choice by the build script,
    // and the user must make a decision in order to create a release build.
    const optimize = b.standardOptimizeOption(.{});

    const exe = b.addExecutable(.{
        .name = "zig-hello-world",
        .root_source_file = .{ .path = "src/main.zig" },
        .target = target,
        .optimize = optimize,
    });
    b.installArtifact(exe);

    const run_exe = b.addRunArtifact(exe);
    const run_step = b.step("run", "Run the application");
    run_step.dependOn(&run_exe.step);

    const test_step = b.step("test", "Run unit tests");
    const unit_tests = b.addTest(.{ .root_source_file = .{ .path = "src/main.zig" }, .target = target, .optimize = optimize });
    const run_unit_tests = b.addRunArtifact(unit_tests);
    test_step.dependOn(&run_unit_tests.step);
}
