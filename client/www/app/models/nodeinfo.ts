const _mode_dir: number = (1 << 32) - 1

export class NodeInfo {
  Id: string;
  Name: string;
  Size: number;
  Mode: number;

  public isDir(): boolean {
    return (this.Mode & _mode_dir) > 0;
  }
}
