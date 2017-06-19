/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.gapid.glviewer.vec;

import java.util.Arrays;

/**
 * A 4x4 double precision matrix.
 */
public class MatD {
  public static final MatD IDENTITY = new MatD();

  private final double[] m;

  /**
   * Creates a new identity matrix.
   */
  public MatD() {
    this(new double[] { 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1 });
  }

  private MatD(double[] m) {
    this.m = m;
  }

  public static MatD of(double[] m) {
    return new MatD(Arrays.copyOf(m, 16));
  }

  public static MatD of(
      double a, double b, double c, double d,
      double e, double f, double g, double h,
      double i, double j, double k, double l,
      double m, double n, double o, double p) {
    return new MatD(new double[] { a, b, c, d, e, f, g, h, i, j, k, l, m, n, o, p });
  }

  public static MatD copyOf(MatD m) {
    return new MatD(Arrays.copyOf(m.m, 16));
  }

  public float[] toFloatArray() {
    return new float[] {
        (float)m[0], (float)m[1], (float)m[2], (float)m[3], (float)m[4], (float)m[5], (float)m[6],
        (float)m[7], (float)m[8], (float)m[9], (float)m[10], (float)m[11], (float)m[12],
        (float)m[13], (float)m[14], (float)m[15]
    };
  }

  /**
   * @param invert Whether to invert the resulting matrix in order to invert the normals.
   * @return The transpose of the inverse of the top 3x3 (i.e. a matrix with just the rotation and
   *     a uniform scale).
   */
  public float[] toNormalMatrix(boolean invert) {
    double a = m[0], d = m[1], g = m[2], b = m[4], e = m[5], h = m[6], c = m[8], f = m[9], k = m[10];
    double ek = e * k, fh = f * h, fg = f * g, dk = d * k, dh = d * h, eg = e * g;
    double det = (invert ? -1 : 1) / (a * (ek - fh) + b * (fg - dk) + c * (dh - eg));
    return new float[] {
      (float)((ek - fh) * det), (float)((c * h - b * k) * det), (float)((b * f - c * e) * det),
      (float)((fg - dk) * det), (float)((a * k - c * g) * det), (float)((c * d - a * f) * det),
      (float)((dh - eg) * det), (float)((b * g - a * h) * det), (float)((a * e - b * d) * det)
    };
  }

  public MatD translate(VecD v) {
    return translate(v.x, v.y, v.z);
  }

  public MatD translate(double tx, double ty, double tz) {
    double[] r = m.clone();
    r[12] = m[0] * tx + m[4] * ty + m[8] * tz;
    r[13] = m[1] * tx + m[5] * ty + m[9] * tz;
    r[14] = m[2] * tx + m[6] * ty + m[10] * tz;
    return new MatD(r);
  }

  public MatD scale(double s) {
    return scale(s, s, s);
  }

  public MatD scale(double sx, double sy, double sz) {
    return multiply(makeScale(sx, sy, sz));
  }

  public MatD scale(VecD v) {
    return scale(v.x, v.y, v.z);
  }

  public double[] inverseOfTop3x3() {
    double a = m[0], d = m[1], g = m[2], b = m[4], e = m[5], h = m[6], c = m[8], f = m[9], k = m[10];
    double ek = e * k, fh = f * h, fg = f * g, dk = d * k, dh = d * h, eg = e * g;
    double det = 1 / (a * (ek - fh) + b * (fg - dk) + c * (dh - eg));
    return new double[] {
      (ek - fh) * det, (fg - dk) * det, (dh - eg) * det,
      (c * h - b * k) * det, (a * k - c * g) * det, (b * g - a * h) * det,
      (b * f - c * e) * det, (c * d - a * f) * det, (a * e - b * d) * det };
  }

  public MatD multiply(MatD mat) {
    return multiply(mat.m);
  }

  public MatD multiply(double[] n) {
    double a00 = m[ 0], a01 = m[ 1], a02 = m[ 2], a03 = m[ 3];
    double a10 = m[ 4], a11 = m[ 5], a12 = m[ 6], a13 = m[ 7];
    double a20 = m[ 8], a21 = m[ 9], a22 = m[10], a23 = m[11];
    double a30 = m[12], a31 = m[13], a32 = m[14], a33 = m[15];
    double b00 = n[ 0], b01 = n[ 1], b02 = n[ 2], b03 = n[ 3];
    double b10 = n[ 4], b11 = n[ 5], b12 = n[ 6], b13 = n[ 7];
    double b20 = n[ 8], b21 = n[ 9], b22 = n[10], b23 = n[11];
    double b30 = n[12], b31 = n[13], b32 = n[14], b33 = n[15];

    return new MatD(new double[] {
      b00 * a00 + b01 * a10 + b02 * a20 + b03 * a30,
      b00 * a01 + b01 * a11 + b02 * a21 + b03 * a31,
      b00 * a02 + b01 * a12 + b02 * a22 + b03 * a32,
      b00 * a03 + b01 * a13 + b02 * a23 + b03 * a33,
      b10 * a00 + b11 * a10 + b12 * a20 + b13 * a30,
      b10 * a01 + b11 * a11 + b12 * a21 + b13 * a31,
      b10 * a02 + b11 * a12 + b12 * a22 + b13 * a32,
      b10 * a03 + b11 * a13 + b12 * a23 + b13 * a33,
      b20 * a00 + b21 * a10 + b22 * a20 + b23 * a30,
      b20 * a01 + b21 * a11 + b22 * a21 + b23 * a31,
      b20 * a02 + b21 * a12 + b22 * a22 + b23 * a32,
      b20 * a03 + b21 * a13 + b22 * a23 + b23 * a33,
      b30 * a00 + b31 * a10 + b32 * a20 + b33 * a30,
      b30 * a01 + b31 * a11 + b32 * a21 + b33 * a31,
      b30 * a02 + b31 * a12 + b32 * a22 + b33 * a32,
      b30 * a03 + b31 * a13 + b32 * a23 + b33 * a33
    });
  }

  public VecD multiply(VecD vec) {
    double x = vec.x, y = vec.y, z = vec.z;
    return new VecD(
      x * m[0] + y * m[4] + z * m[8] + m[12],
      x * m[1] + y * m[5] + z * m[9] + m[13],
      x * m[2] + y * m[6] + z * m[10] + m[14]
    );
  }

  public void multiply(double[] vecIn, int inOffset, double[] vecOut, int outOffset) {
    double x = vecIn[inOffset + 0], y = vecIn[inOffset + 1], z = vecIn[inOffset + 2];
    vecOut[outOffset + 0] = x * m[0] + y * m[4] + z * m[8] + m[12];
    vecOut[outOffset + 1] = x * m[1] + y * m[5] + z * m[9] + m[13];
    vecOut[outOffset + 2] = x * m[2] + y * m[6] + z * m[10] + m[14];
  }

  public static MatD projection(double width, double height, double focalLength, double zNear) {
    double scale = 2 * focalLength;
    return new MatD(new double[] {
      scale / width, 0, 0, 0,
      0, scale / height, 0, 0,
      0, 0, -1, -1,
      0, 0, -2 * zNear, 0
    });
  }

  public static MatD lookAt(VecD eye, VecD center, VecD up) {
    VecD forward = center.subtract(eye).normalize();
    VecD side = forward.cross(up).normalize();
    VecD newUp = side.cross(forward).normalize();
    return new MatD(new double[] {
      side.x, newUp.x, -forward.x, 0,
      side.y, newUp.y, -forward.y, 0,
      side.z, newUp.z, -forward.z, 0,
      -eye.x * side.x    - eye.y * side.y    - eye.z * side.z,
      -eye.x * newUp.x   - eye.y * newUp.y   - eye.z * newUp.z,
       eye.x * forward.x + eye.y * forward.y + eye.z * forward.z,
      1
    });
  }

  public static MatD translation(double tx, double ty, double tz) {
    return new MatD(new double[] {
      1, 0, 0, 0,
      0, 1, 0, 0,
      0, 0, 1, 0,
      tx, ty, tz, 1
    });
  }

  /**
   * Creates a matrix that is a translation followed by a rotation about the X and Y axis. Note,
   * the angles are in degrees.
   */
  public static MatD makeTranslationRotXY(
      double tx, double ty, double tz, double angleX, double angleY) {
    double cosX = Math.cos(Math.toRadians(angleX));
    double sinX = Math.sin(Math.toRadians(angleX));
    double cosY = Math.cos(Math.toRadians(angleY));
    double sinY = Math.sin(Math.toRadians(angleY));
    return new MatD(new double[] {
      cosY, sinX * sinY, -cosX * sinY, 0,
      0, cosX, sinX, 0,
      sinY, -sinX * cosY, cosX * cosY, 0,
      tx, ty, tz, 1
    });
  }

  /**
   * Creates a scale matrix.
   */
  public static MatD makeScale(double scale) {
    return new MatD(new double[] {
        scale, 0, 0, 0,
        0, scale, 0, 0,
        0, 0, scale, 0,
        0, 0, 0, 1
    });
  }

  /**
   * Creates a scale matrix.
   */
  public static MatD makeScale(double sx, double sy, double sz) {
    return new MatD(new double[] {
        sx, 0, 0, 0,
        0, sy, 0, 0,
        0, 0, sz, 0,
        0, 0, 0, 1
    });
  }

  /**
   * Creates a scale matrix.
   */
  public static MatD makeScale(VecD v) {
    return makeScale(v.x, v.y, v.z);
  }

  /**
   * Creates a matrix that is a scale followed by a translation.
   */
  public static MatD makeScaleTranslation(double scale, VecD t) {
    return new MatD(new double[] {
      scale, 0, 0, 0,
      0, scale, 0, 0,
      0, 0, scale, 0,
      scale * t.x, scale * t.y, scale * t.z, 1
    });
  }

  /**
   * Creates a matrix that is a Z-up to Y-up rotation followed by a scale then translation.
   */
  public static MatD makeScaleTranslationZupToYup(double scale, VecD t) {
    return new MatD(new double[] {
      scale, 0, 0, 0,
      0, 0, -scale, 0,
      0, scale, 0, 0,
      scale * t.x, scale * t.z, -scale * t.y, 1
    });
  }
}
