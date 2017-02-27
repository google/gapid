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
package com.google.gapid.lang.glsl;

import com.google.common.collect.ImmutableSet;
import com.google.gapid.widgets.Theme;

import org.eclipse.jface.text.Document;
import org.eclipse.jface.text.IDocument;
import org.eclipse.jface.text.IDocumentExtension3;
import org.eclipse.jface.text.IDocumentPartitioner;
import org.eclipse.jface.text.ITextDoubleClickStrategy;
import org.eclipse.jface.text.ITextViewer;
import org.eclipse.jface.text.TextAttribute;
import org.eclipse.jface.text.presentation.IPresentationReconciler;
import org.eclipse.jface.text.presentation.PresentationReconciler;
import org.eclipse.jface.text.rules.DefaultDamagerRepairer;
import org.eclipse.jface.text.rules.EndOfLineRule;
import org.eclipse.jface.text.rules.FastPartitioner;
import org.eclipse.jface.text.rules.ICharacterScanner;
import org.eclipse.jface.text.rules.IPredicateRule;
import org.eclipse.jface.text.rules.IRule;
import org.eclipse.jface.text.rules.IToken;
import org.eclipse.jface.text.rules.ITokenScanner;
import org.eclipse.jface.text.rules.IWordDetector;
import org.eclipse.jface.text.rules.MultiLineRule;
import org.eclipse.jface.text.rules.NumberRule;
import org.eclipse.jface.text.rules.RuleBasedPartitionScanner;
import org.eclipse.jface.text.rules.RuleBasedScanner;
import org.eclipse.jface.text.rules.Token;
import org.eclipse.jface.text.rules.WordPatternRule;
import org.eclipse.jface.text.rules.WordRule;
import org.eclipse.jface.text.source.ISourceViewer;
import org.eclipse.jface.text.source.SourceViewerConfiguration;
import org.eclipse.swt.SWT;

import java.util.Set;

/**
 * {@link SourceViewerConfiguration} to use for syntax highlighting of GLSL code.
 */
public class GlslSourceConfiguration extends SourceViewerConfiguration {
  private static final String PARTIONING = GlslSourceConfiguration.class.getName();
  private static final String SOURCE = IDocument.DEFAULT_CONTENT_TYPE;
  private static final String SINGLE_COMMENT = "SINGLE_COMMENT";
  private static final String MULTI_COMMENT = "MULTI_COMMENT";
  private static final String[] CONTENT_TYPES = {
      SOURCE, SINGLE_COMMENT, MULTI_COMMENT
  };

  private static final IWordDetector WORD_DETECTOR = new IWordDetector() {
    @Override
    public boolean isWordStart(char c) {
      return Character.isJavaIdentifierStart(c);
    }

    @Override
    public boolean isWordPart(char c) {
      return Character.isJavaIdentifierPart(c);
    }
  };

  private static final String[] KEYWORDS = {
      "attribute", "const", "uniform", "buffer", "shared", "coherent", "volatile", "restrict",
      "readonly", "writeonly", "atomic_uint", "layout", "centroid", "flat", "smooth", "patch",
      "sample", "precise", "break", "continue", "do", "for", "while", "switch", "case", "default",
      "if", "else", "in", "out", "inout", "float", "int", "void", "bool", "true", "false",
      "invariant", "discard", "return", "mat2", "mat3", "mat4", "mat2x2", "mat2x3", "mat2x4",
      "mat3x2", "mat3x3", "mat3x4", "mat4x2", "mat4x3", "mat4x4", "vec2", "vec3", "vec4", "ivec2",
      "ivec3", "ivec4", "bvec2", "bvec3", "bvec4", "uint", "uvec2", "uvec3", "uvec4", "lowp",
      "mediump", "highp", "precision", "sampler2D", "sampler3D", "samplerCube", "sampler2DShadow",
      "samplerCubeShadow", "sampler2DArray", "sampler2DArrayShadow", "isampler2D", "isampler3D",
      "isamplerCube", "isampler2DArray", "usampler2D", "usampler3D", "usamplerCube",
      "usampler2DArray", "sampler2DMS", "isampler2DMS", "usampler2DMS", "samplerBuffer",
      "isamplerBuffer", "usamplerBuffer", "imageBuffer", "iimageBuffer", "uimageBuffer",
      "imageCubeArray", "iimageCubeArray", "uimageCubeArray", "samplerCubeArray",
      "isamplerCubeArray", "usamplerCubeArray", "samplerCubeArrayShadow", "sampler2DMSArray",
      "isampler2DMSArray", "usampler2DMSArray", "image2DArray", "iimage2DArray", "uimage2DArray",
      "image2D", "iimage2D", "uimage2D", "image3D", "iimage3D", "uimage3D", "imageCube",
      "iimageCube", "uimageCube", "struct", "varying"
  };

  protected static final Set<String> PREPROCESSOR_KEYWORDS = ImmutableSet.of(
      "define", "undef", "if", "ifdef", "ifndef", "else", "elif", "endif", "error", "pragma",
      "extension", "version", "line"
  );


  private final Theme theme;

  public GlslSourceConfiguration(Theme theme) {
    this.theme = theme;
  }

  public static IDocument createDocument(String text) {
    Document document = new Document(text);
    setup(document);
    return document;
  }

  public static <D extends IDocument & IDocumentExtension3> void setup(D document) {
    IDocumentPartitioner partitioner = createPartioner();
    ((IDocumentExtension3)document).setDocumentPartitioner(PARTIONING, partitioner);
    partitioner.connect(document);
  }

  private static IDocumentPartitioner createPartioner() {
    RuleBasedPartitionScanner scanner = new RuleBasedPartitionScanner();
    scanner.setPredicateRules(new IPredicateRule[] {
        new EndOfLineRule("//", new Token(SINGLE_COMMENT), (char)0),
        new MultiLineRule("/*", "*/", new Token(MULTI_COMMENT), (char)0, true)
    });
    return new FastPartitioner(scanner, CONTENT_TYPES);
  }

  @Override
  public String getConfiguredDocumentPartitioning(ISourceViewer sourceViewer) {
    return PARTIONING;
  }

  @Override
  public String[] getConfiguredContentTypes(ISourceViewer sourceViewer) {
    return CONTENT_TYPES;
  }

  @Override
  public IPresentationReconciler getPresentationReconciler(ISourceViewer sourceViewer) {
    PresentationReconciler reconciler= new PresentationReconciler();
    reconciler.setDocumentPartitioning(getConfiguredDocumentPartitioning(sourceViewer));

    addDamagerRepairer(reconciler, createCommentScanner(), SINGLE_COMMENT);
    addDamagerRepairer(reconciler, createCommentScanner(), MULTI_COMMENT);
    addDamagerRepairer(reconciler, createCodeScanner(), SOURCE);
    return reconciler;
  }

  @Override
  public ITextDoubleClickStrategy getDoubleClickStrategy(ISourceViewer viewer, String type) {
    // Avoid dependency on ICU.
    return new ITextDoubleClickStrategy() {
      @Override
      public void doubleClicked(ITextViewer v) {
        // TODO
      }
    };
  }

  private static void addDamagerRepairer(
      PresentationReconciler reconciler, ITokenScanner scanner, String type) {
    DefaultDamagerRepairer commentDamagerRepairer = new DefaultDamagerRepairer(scanner);
    reconciler.setDamager(commentDamagerRepairer, type);
    reconciler.setRepairer(commentDamagerRepairer, type);
  }

  private ITokenScanner createCommentScanner() {
    RuleBasedScanner scanner = new RuleBasedScanner();
    scanner.setDefaultReturnToken(new Token(new TextAttribute(theme.commentColor())));
    return scanner;
  }

  private ITokenScanner createCodeScanner() {
    RuleBasedScanner scanner = new RuleBasedScanner();
    scanner.setRules(new IRule[] {
        createKeywordRule(),
        createGlIdentifierRule(),
        createIdentifierRule(),
        createNumberRule(),
        createPreprocessorRul()
    });
    return scanner;
  }

  private IRule createKeywordRule() {
    Token token = new Token(new TextAttribute(theme.keywordColor()));
    WordRule rule = new WordRule(WORD_DETECTOR, Token.UNDEFINED, false);
    for (String keyword : KEYWORDS) {
      rule.addWord(keyword, token);
    }
    return rule;
  }

  private IRule createGlIdentifierRule() {
    return new WordPatternRule(WORD_DETECTOR, "gl_", null,
        new Token(new TextAttribute(theme.identifierColor(), null, SWT.BOLD)), (char)-1);
  }

  private IRule createIdentifierRule() {
    WordRule rule = new WordRule(WORD_DETECTOR, new Token(null), false);
    rule.addWord("main", new Token(new TextAttribute(theme.identifierColor(), null, SWT.BOLD)));
    return rule;
  }

  private IRule createNumberRule() {
    return new NumberRule(new Token(new TextAttribute(theme.numericConstantColor())));
  }

  private IRule createPreprocessorRul() {
    return new PreprocessorRule(new Token(new TextAttribute(theme.preprocessorColor())));
  }

  private static class PreprocessorRule implements IRule {
    private final StringBuilder buffer = new StringBuilder();
    private final Token token;

    public PreprocessorRule(Token token) {
      this.token = token;
    }

    @Override
    public IToken evaluate(ICharacterScanner scanner) {
      if (scanner.getColumn() != 0) {
        // Wait for line start
        return Token.UNDEFINED;
      }

      // Ignore any whitespace at the beginning.
      int count = 0;
      int c;
      do {
        c = scanner.read();
        count++;
      } while (c != ICharacterScanner.EOF && Character.isWhitespace(c) && c != '\r' && c != '\n');

      if (c == '#') {
        count = 0; // Everything up to here will be marked as a preprocessor rule for sure.

        // Ignore whitespace between '#' and token.
        do {
          c = scanner.read();
          count++;
        } while (c != ICharacterScanner.EOF && Character.isWhitespace(c) && c != '\r' && c != '\n');

        if (c != ICharacterScanner.EOF && Character.isAlphabetic(c)) {
          buffer.setLength(0);
          do {
            buffer.append((char)c);
            c = scanner.read();
            count++;
          } while (c != ICharacterScanner.EOF && Character.isAlphabetic(c));
          scanner.unread();

          if (PREPROCESSOR_KEYWORDS.contains(buffer.toString())) {
            return token;
          }
          count--;
        }
        for (; count > 0; count--) {
          scanner.unread();
        }
        return token;
      }
      for (; count > 0; count--) {
        scanner.unread();
      }
      return Token.UNDEFINED;
    }
  }
}
