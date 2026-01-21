/**
 * Path validation utilities for secure file access
 * Only allows paths within the configured root directory
 */

import * as fs from 'fs';
import * as path from 'path';

/**
 * Validates and resolves a user-provided path
 * @param userPath - The path provided by the user (must be relative)
 * @param rootPath - The root path that all access must be within
 * @returns The resolved absolute path
 * @throws Error if path is invalid or outside root
 */
export function validatePath(userPath: string, rootPath: string): string {
  // Check for null bytes
  if (userPath.includes('\0')) {
    throw new Error('Invalid path: contains null bytes');
  }

  // Check for absolute paths
  if (path.isAbsolute(userPath)) {
    throw new Error('Absolute paths are not allowed');
  }

  // Resolve the path
  const resolved = path.resolve(rootPath, userPath);
  
  // Get real paths (resolves symlinks)
  let real: string;
  let realRoot: string;
  
  try {
    realRoot = fs.realpathSync(rootPath);
  } catch (err) {
    throw new Error(`Root path does not exist: ${rootPath}`);
  }
  
  try {
    real = fs.realpathSync(resolved);
  } catch (err) {
    // Path doesn't exist, check parent directory
    const parent = path.dirname(resolved);
    try {
      const realParent = fs.realpathSync(parent);
      if (!realParent.startsWith(realRoot + path.sep) && realParent !== realRoot) {
        throw new Error('Path outside root directory');
      }
      // Parent is valid, return the resolved path
      return resolved;
    } catch {
      throw new Error(`Path does not exist: ${userPath}`);
    }
  }
  
  // Check if resolved path is within root
  if (!real.startsWith(realRoot + path.sep) && real !== realRoot) {
    throw new Error('Path outside root directory');
  }
  
  return real;
}

/**
 * Check if a path exists
 */
export function pathExists(filePath: string): boolean {
  return fs.existsSync(filePath);
}
