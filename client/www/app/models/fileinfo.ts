export class FileInfo {
  Id: string;
  Name: string;
  Size: number;
  Mode: number;

  isDir(): boolean {
    return this.Mode > 0;
  }
}
