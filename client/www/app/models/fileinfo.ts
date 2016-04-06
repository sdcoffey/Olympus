export class FileInfo {
  Id: string;
  Name: string;
  Size: number;
  Mode: number;

  public isDir(): boolean {
    return this.Mode > 0;
  }
}
